package cache

import (
	"fmt"
	"g2cache/app/layer/cache/first"
	"g2cache/app/layer/cache/interface"
	"g2cache/app/layer/cache/sencond"
	"g2cache/app/layer/container"
	"g2cache/app/layer/entity"
	"g2cache/app/layer/helper"
	"g2cache/app/layer/json"
	"g2cache/app/layer/lock"
	"g2cache/app/layer/pool"
	"g2cache/app/layer/pubsub"
	"g2cache/app/layer/task"
	"github.com/gogf/gf/os/glog"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"sync"
	"time"
)

type LayeringCache struct {
	//本地高速缓存
	freecache *first.FreeCache
	//redis缓存
	redis *sencond.RedisCache
	//redis分布式锁
	redisLock *lock.RedisLock
	//缓存主动在失效前强制刷新缓存的时间
	preloadTime int
	//是否强制刷新（走数据库），默认是false
	forceRefresh bool
	//一级缓存过期时间
	firstExpireTime int
	//二级缓存过期时间
	sencondExpireTime int
	//线程池
	pool     *pool.Pool
	stop     chan struct{}
	stopOnce sync.Once
	tcn      *container.ThreadContainer
}

func init() {
	helper.InitRedis()
	container.InitThreadTimeout()
	pubsub.InitPubSub()
	task.InitTask()
}


//创建一个分布式缓存
func NewCache(preloadTime, firstExpireTime, sencondExpireTime int, forceRefresh bool) (layerCache *LayeringCache, err error) {
	//创建一级缓存
	freecache := first.NewFirstCache()
	//创建二级缓存
	redisCache, _ := sencond.NewRedisCache()
	//创建redis分布式锁
	redisLock := lock.NewRedisLock(redisCache)
	//创建线程容器
	tcn := container.NewThreadContainer()
	//创建线程池
	pool := pool.NewPool(helper.DefaultGPoolWorkerNum, helper.DefaultGPoolJobQueueChanLen)
	layerCache = &LayeringCache{
		freecache:         freecache,
		redis:             redisCache,
		redisLock:         redisLock,
		preloadTime:       preloadTime,
		forceRefresh:      forceRefresh,
		firstExpireTime:   firstExpireTime,
		sencondExpireTime: sencondExpireTime,
		tcn:               tcn,
		pool:              pool,
		stop:              make(chan struct{}, 1),
		stopOnce:          sync.Once{},
	}
	//todo 开启监控
	layerCache.monitor()
	return layerCache, nil
}
func (g *LayeringCache) monitor() {
	//t := time.NewTicker(time.Duration(CacheMonitorSecond) * time.Second)
	//for {
	//	select {
	//	case <-g.stop:
	//		return
	//	case <-t.C:
	//
	//	}
	//}
}

//根据key获取value，缓存不存在时，获取数据库
func (g *LayeringCache) Get(key string, obj interface{}, loadFn _interface.LoadDataSourceFunc) (interface{}, error) {
	select {
	case <-g.stop:
		return nil, helper.CacheClose
	default:

	}
	//参数校验
	if key == "" {
		return nil, helper.KeyNotEmpty
	}
	if obj == nil {
		return nil, helper.ObjNotEmpty
	}
	if loadFn == nil {
		return nil, helper.LoadDataNotEmpty
	}

	var resultObj *entity.CacheEntity
	//查询一级缓存
	result, ok, _ := g.freecache.Get(key, resultObj)
	if ok && result != nil {
		glog.Infof("freecache 命中一级缓存 key=%s,result=%s", key, json.ToJson(result))
		return g.getResult(result), nil
	}
	//查询二级缓存
	result, ok, _ = g.redis.Get(key, resultObj)
	if result == nil {
		//查询数据库
		result = g.executeCacheMethod(key, obj, loadFn)
	} else {
		//缓存预刷新
		g.refreshCache(key, result.Value, loadFn)
	}
	//设置一级缓存
	g.freecache.SetExpireTime(key, result, time.Duration(g.firstExpireTime))
	glog.Infof("查询二级缓存,并将数据放到一级缓存。 key=%s,返回值是:%s", key, json.ToJson(result))
	return g.getResult(result), nil
}

//设置key value ttl
func (g *LayeringCache) Set(key string, obj interface{}) error {
	select {
	case <-g.stop:
		return helper.CacheClose
	default:

	}
	//参数校验
	if key == "" {
		return helper.KeyNotEmpty
	}
	if obj == nil {
		return helper.ObjNotEmpty
	}
	value := entity.NewEntity(obj, g.firstExpireTime, g.sencondExpireTime)
	g.setCache(key, value)
	metaDump, _ := jsoniter.MarshalToString(value)

	glog.Infof("设置缓存成功，key:%s , value: %s\n", key, metaDump)
	return nil
}

//查询数据库
func (g *LayeringCache) executeCacheMethod(key string, obj interface{}, loadFn _interface.LoadDataSourceFunc) *entity.CacheEntity {
	lock := &sync.Mutex{}
	cond := sync.NewCond(lock)
	cond.L.Lock()
	var resultLoad *entity.CacheEntity
	for true {
		var resultObj *entity.CacheEntity
		resultObj.Value = obj
		//查询一级缓存
		result, ok, _ := g.freecache.Get(key, resultObj)
		if ok && result != nil {
			resultLoad = result
			glog.Infof("redis缓存 key= %s 获取到锁后查询一级缓存命中，不需要执行查询DB的方法", key)
			break
		}
		//获取分布式锁
		defer g.redisLock.UnLock(key)
		if g.redisLock.Lock(key, 5) {
			dbresult, _ := loadFn(key)
			//如果数据库为nil,赋予默认值，给个超时时间10s
			if dbresult == nil {
				defaultValue := &entity.CacheEntity{
					Value:             new(helper.NullValue),
					FirstExpireTime:   5,
					SencondExpireTime: 10,
				}
				g.setCache(key, defaultValue)
				resultLoad = defaultValue
				break
			}
			helper.Clone(dbresult, resultObj)
			//使用默认过期时间
			resultObj.FirstExpireTime = g.firstExpireTime
			resultObj.SencondExpireTime = g.sencondExpireTime
			g.setCache(key, resultObj)
			glog.Infof("缓存 key= %s 从数据库获取数据完毕并放入一级缓存二级缓存，唤醒所有等待线程", key)
			//通知释放资源
			g.tcn.NotifyAll(key)
			resultLoad = resultObj
			break
		} else {
			glog.Infof("缓存 key= %s 从数据库获取数据未获取到锁，进入等待状态!", key)
			//线程阻塞 500ms
			g.tcn.Await(key, cond, 500)
		}
	}
	cond.L.Unlock()
	return resultLoad
}

//预刷新缓存
func (g *LayeringCache) refreshCache(key string, result interface{}, loadFn _interface.LoadDataSourceFunc) {
	//异步刷新
	g.pool.SendJob(func() {
		//校验是否需要刷新
		if g.isRefresh(key) {
			//校验是否强制刷新
			if !g.forceRefresh {
				//软刷新
				glog.Infof("redis缓存 key=%s 软刷新缓存模式", key)
				g.softRefresh(key)
			} else {
				//强制刷新
				g.forceRefreshDB(key, result, loadFn)
			}
		}
	})
}

//软刷新
func (g *LayeringCache) softRefresh(key string) {
	keys := fmt.Sprintf(helper.LayeringTermExpireTime, key)
	if g.redisLock.Lock(keys, 5) {
		g.redis.ExpireTime(key, time.Duration(g.sencondExpireTime))
	}
	defer g.redisLock.UnLock(key)

}

//硬刷新 查询数据库
func (g *LayeringCache) forceRefreshDB(key string, result interface{}, loadFn _interface.LoadDataSourceFunc) {
	keys := fmt.Sprintf(helper.LayeringExecuteDBTime, key)
	if g.redisLock.Lock(keys, 5) {
		dbresult, _ := loadFn(key)
		if dbresult != result {
			value := entity.NewEntity(dbresult, g.firstExpireTime, g.sencondExpireTime)
			g.setCache(key, value)
		}
	}

	defer g.redisLock.UnLock(key)

}

//校验是否需要刷新缓存
func (g *LayeringCache) isRefresh(key string) bool {
	ttl, _ := g.redis.TTL(key)
	if ttl < 0 {
		return true
	}
	if ttl < g.preloadTime {
		return true
	}
	return false
}

//设置缓存
func (g *LayeringCache) setCache(key string, resultObj *entity.CacheEntity) {
	g.freecache.SetExpireTime(key, resultObj, time.Duration(resultObj.FirstExpireTime))
	g.redis.SetExpireTime(key, resultObj, time.Duration(resultObj.SencondExpireTime))
	data := &_interface.ChannelMetedata{
		Key:    key,
		Gid:    uuid.New().String(),
		Action: helper.SetPublishType,
		Data:   resultObj,
	}
	g.pool.SendJob(func() {
		msgQueuekey := fmt.Sprintf(helper.LayeringMsgQueueKey, "node1")
		g.redis.LPush(msgQueuekey, data)
		g.redis.Publish(data)
	})
}

//获取结果
func (g *LayeringCache) getResult(result *entity.CacheEntity) interface{} {
	_, ok := result.Value.(helper.NullValue)
	if ok {
		return nil
	}
	return result.Value
}

//todo
//
//func (g *LayeringCache) close() {
//	if g.stop != nil {
//		close(g.stop)
//	}
//	if g.redis != nil {
//		g.redis.Close()
//	}
//	if g.freecache != nil {
//		g.freecache.Close()
//	}
//
//}
//
