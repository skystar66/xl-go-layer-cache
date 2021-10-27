package task

import (
	"fmt"
	"g2cache/app/bizseer/cache/sencond"
	"g2cache/app/bizseer/helper"
	"g2cache/app/bizseer/lock"
	"g2cache/app/bizseer/pool"
	"g2cache/app/bizseer/pubsub"
	sync2 "g2cache/app/bizseer/sync"
	"github.com/gogf/gf/os/glog"
	"github.com/robfig/cron"
	"sync"
	"sync/atomic"
	"time"
)



type CacheTask struct {
	//线程池
	pool       *pool.Pool
	stop       chan struct{}
	stopOnce   sync.Once
	rediscache *sencond.RedisCache
	//redis分布式锁
	redisLock *lock.RedisLock
	pubsubs *pubsub.PubSubService
	syncdata *sync2.SyncDataService


}

//系统初始化
func InitTask() {
	cacheTask:=NewCacheTask()
	//服务启动同步最新的消息偏移量OFFSET
	cacheTask.SyncOffset()
	//启动拉取消息任务线程,防止丢消息
	go cacheTask.taskPullMsg()
	//启动凌晨重置消息队列任务线程
	go cacheTask.clearMessageQueueTask()
	//重连检测，防止redis,pub ,sub 掉线
	go cacheTask.taskReconnection()
	glog.Info("task 任务初始化成功！！！")

}

var (
	onceinstance *CacheTask
	once         sync.Once
)

func NewCacheTask()*CacheTask {
	pool := pool.NewPool(helper.DefaultGPoolWorkerNum, helper.DefaultGPoolJobQueueChanLen)
	pubsubs:=pubsub.NewPubSub()
	redisCache,_:= sencond.NewRedisCache()
	redislock:=lock.NewRedisLock(redisCache)
	syncdataservice:=sync2.NewSyncDataService()
	once.Do(func() {
		cacheTask :=CacheTask{
			pool:       pool,
			stop:       make(chan struct{}, 1),
			stopOnce:   sync.Once{},
			rediscache: redisCache,
			redisLock:  redislock,
			pubsubs:    pubsubs,
			syncdata: syncdataservice,
		}
		onceinstance = &cacheTask
	})
	return onceinstance
}


//服务启动同步最新的消息偏移量OFFSET
func (c *CacheTask) SyncOffset() {
	key := fmt.Sprintf(helper.LayeringMsgQueueKey, "node1")
	offset, _ := c.rediscache.Llen(key)
	atomic.SwapInt64(&pubsub.OFFSET, offset-1)

	glog.Infof("同步 OFFSET:【%d】 成功", offset-1)
}

//todo 试一下 每2s一次
//清理消息队列，防止消息堆积，增加消息一致性，重置本地OFFSET
func (c *CacheTask) clearMessageQueueTask() {
	key := fmt.Sprintf(helper.LayeringMsgQueueKey, "node1")
	cron := cron.New()
	err := cron.AddFunc(helper.Cron_Clean_Message_Queue, func() {
		if c.redisLock.Lock(key, 5) {
			c.rediscache.Evict(key)
		}
		// 重置偏移量，其它服务器也会更新
		atomic.SwapInt64(&pubsub.OFFSET, -1)
		defer c.redisLock.UnLock(key)
	})
	if err != nil {
		fmt.Errorf("AddFunc error : %v", err)
		return
	}
	cron.Start()
	defer cron.Stop()
	//select {}
}

func (c *CacheTask) taskReconnection() {
	t := time.NewTicker(time.Duration(helper.PULL_MSG_TIME_SENCOND) * time.Second)
	for {
		select {
		case <-c.stop:
			//关闭定时器
			t.Stop()
			return
		case <-t.C:
			c.reconnection()
		}
	}
}

//启动重连检测，防止redis,pub ,sub 掉线
func (c *CacheTask) reconnection() {

	times := time.Now().Unix() - atomic.LoadInt64(&helper.LAST_PUSH_TIME)
	if times >= helper.RECONNECTION_TIME {
		atomic.SwapInt64(&helper.LAST_PUSH_TIME, time.Now().Unix())
		//  redis pub/sub 监听器,重新发起订阅
		c.pubsubs.Subscribe()
	}
}

//每30s 执行一次拉取消息的任务
func (c *CacheTask) taskPullMsg() {
	t := time.NewTicker(time.Duration(helper.PULL_MSG_TIME_SENCOND) * time.Second)
	for {
		select {
		case <-c.stop:
			//关闭定时器
			t.Stop()
			return
		case <-t.C:
			c.pullMsg()
		}
	}
}

//拉取redis数据
func (t *CacheTask) pullMsg() {
	//获取redis的最新的offset
	key := fmt.Sprintf(helper.LayeringMsgQueueKey, "node1")
	num, _ := t.rediscache.Llen(key)
	maxOffset := num - 1
	oldOffset := atomic.SwapInt64(&pubsub.OFFSET, maxOffset)
	if oldOffset != 0 && oldOffset >= maxOffset {
		/**本地已是最新同步的缓存信息啦*/
		return
	}
	endOffset := maxOffset - oldOffset - 1
	/**获取消息*/
	result, _ := t.rediscache.LRange(key, 0, endOffset)
	if len(result) <= 0 {
		return
	}
	for _, data := range result {
		glog.Infof("【缓存同步】redis 通过PULL方式处理本地缓存，startOffset:【0】,endOffset:【{}】,消息内容：{}", endOffset, data)
		t.syncdata.SyncData(&data)
	}
}
