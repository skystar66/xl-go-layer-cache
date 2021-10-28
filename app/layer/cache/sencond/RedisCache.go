package sencond

import (
	"fmt"
	"g2cache/app/layer/cache/interface"
	"g2cache/app/layer/entity"
	"g2cache/app/layer/helper"
	g2cache "g2cache/app/layer/log"
	"github.com/gogf/gf/os/glog"
	"github.com/gomodule/redigo/redis"
	jsoniter "github.com/json-iterator/go"
	"sync"
	"time"
)

//sencond pub sub channel
var DefaultPubSubRedisChannel = "g2cache-pubsub-channel"

type RedisCache struct {
	pool       *redis.Pool
	pubsubPool *redis.Pool
	stop       chan struct{}
	stopOnce   sync.Once
}

var (
	onceinstance *RedisCache
	once         sync.Once
)

//创建redis缓存
func NewRedisCache() (*RedisCache, error) {
	pool, err := helper.GetRedisPool()
	if err != nil {
		g2cache.LogErrF("redis pool init err %v", err)
		return nil, fmt.Errorf("redis pool init err %v", err)
	}
	pubsubPool, err := helper.GetRedisPool()
	if err != nil {
		return nil, fmt.Errorf("redis pub sub pool init err %v", err)
	}
	//单例
	once.Do(func() {
		redisCache := RedisCache{
			pool:       pool,
			pubsubPool: pubsubPool,
			stop:       make(chan struct{}, 1),
			stopOnce:   sync.Once{},
		}
		onceinstance = &redisCache
	})
	return onceinstance, nil
}

func (c *RedisCache) Get(key string, resultObj interface{}) (*entity.CacheEntity, bool, error) {
	select {
	case <-c.stop:
		return nil, false, helper.RedisStorageClose
	default:

	}
	value, err := helper.RedisGetString(c.pool, key)
	if err != nil {
		return nil, false, err
	}
	var result entity.CacheEntity
	result.Value = resultObj //设置反射类型
	jsoniter.UnmarshalFromString(value, result)
	return &result, true, nil
}

func (c *RedisCache) Set(key string, e *entity.CacheEntity) error {
	select {
	case <-c.stop:
		return helper.RedisStorageClose
	default:

	}
	value, _ := jsoniter.MarshalToString(e)
	err := helper.RedisSetEXString(c.pool, key, value, int64(e.SencondExpireTime))
	return err
}

func (c *RedisCache) SetExpireTime(key string, e *entity.CacheEntity, duration time.Duration) error {
	select {
	case <-c.stop:
		return helper.RedisStorageClose

	default:

	}
	value, _ := jsoniter.MarshalToString(e)
	err := helper.RedisSetEXString(c.pool, key, value, int64(duration.Seconds()))
	return err
}

func (c *RedisCache) ExpireTime(key string, duration time.Duration) error {
	select {
	case <-c.stop:
		return helper.RedisStorageClose
	default:

	}
	helper.Expire(c.pool, key, int64(duration))
	return nil
}

func (c *RedisCache) Evict(key string) error {
	select {
	case <-c.stop:
		return helper.RedisStorageClose
	default:

	}
	err := helper.RedisDel(c.pool, key)
	return err
}

func (c *RedisCache) Llen(key string) (int64, error) {
	select {
	case <-c.stop:
		return 0, helper.RedisStorageClose
	default:

	}
	len, _ := helper.LLen(c.pool, key)
	return len, nil

}
func (c *RedisCache) LRange(key string, startOffset, endOffset int64) ([]_interface.ChannelMetedata, error) {
	select {
	case <-c.stop:
		return nil, helper.RedisStorageClose
	default:

	}
	result, _ := redis.Values(helper.LRange(c.pool, key, startOffset, endOffset))
	metaDatas := []_interface.ChannelMetedata{}
	metaData := &_interface.ChannelMetedata{}
	for _, v := range result {
		helper.Decode(v, nil, metaData)
		metaDatas = append(metaDatas, *metaData)
	}
	return metaDatas, nil
}

func (c *RedisCache) LPush(key string, member interface{}) error {
	select {
	case <-c.stop:
		return helper.RedisStorageClose
	default:
	}

	return helper.LPush(c.pool, key, member)
}

func (c *RedisCache) Clear() {
}

func (c *RedisCache) ThreadSafe() { // Need to ensure thread safety {

}

func (f *RedisCache) close() {
	//关闭缓存
	close(f.stop)
}

func (f *RedisCache) Close() {
	f.stopOnce.Do(f.close)
}

//消息订阅
func (c *RedisCache) Subscribe(dataChan chan<- *_interface.ChannelMetedata) error {
	select {
	case <-c.stop:
		return helper.RedisStorageClose
	default:

	}
	pubsubConn := c.pubsubPool.Get()
	pubsub := redis.PubSubConn{pubsubConn}
	if err := pubsub.Subscribe(DefaultPubSubRedisChannel); err != nil {
		glog.Infof("rds subscribe channel=%v, err=%v\n", err)
		return err
	}
	glog.Infof("rds subscribe channel=%v success! start listener...\n", DefaultPubSubRedisChannel)

Loop:
	for {
		select {
		case <-c.stop:
			return helper.RedisStorageClose
		default:
		}
		switch v := pubsub.Receive().(type) {
		case redis.Message:
			metadata := &_interface.ChannelMetedata{}
			err := jsoniter.Unmarshal(v.Data, metadata)
			if err != nil {
				glog.Infof("rds subscribe Unmarshal data: %v,err:%v\n", metadata, err)
				continue
			}
			select {
			case <-c.stop:
				return helper.RedisStorageClose
			default:
			}
			dataChan <- metadata
		case error:
			glog.Infof("rds subscribe receive error, msg=%v\n", v)
			break Loop
		}
	}
	return nil
}

//消息推送
func (c *RedisCache) Publish(data *_interface.ChannelMetedata) error {
	select {
	case <-c.stop:
		return helper.RedisStorageClose
	default:
	}
	msg, _ := jsoniter.MarshalToString(data)
	helper.RedisPublish(c.pubsubPool, DefaultPubSubRedisChannel, msg)
	glog.Infof("key: %s,channel：%s,msg：%s publish success!", data.Key, DefaultPubSubRedisChannel, msg)

	return nil
}

func (c *RedisCache) Lock(key, script string, ttlSencond, kecount int) (bool, error) {
	luaLockScript := redis.NewScript(kecount, script)
	conn, _ := helper.GetRedisConn(c.pool)
	lockResult, _ := luaLockScript.Do(conn, key, ttlSencond)
	result, _ := redis.Int(lockResult, nil)
	if result != 1 {
		return false, nil
	}
	return true, nil

}

func (c *RedisCache) Unlock(key, script string) (bool, error) {
	luaLockScript := redis.NewScript(1, script)
	conn, _ := helper.GetRedisConn(c.pool)
	luaLockScript.Do(conn, key)
	return true, nil

}

func (c *RedisCache) RenewalExpiretime(key, script string, ttlSencond int) (bool, error) {
	luaLockScript := redis.NewScript(1, script)
	conn, _ := helper.GetRedisConn(c.pool)
	lockResult, _ := luaLockScript.Do(conn, key, ttlSencond)
	result, _ := redis.Int(lockResult, nil)
	if result != 1 {
		return false, nil
	}
	return true, nil
}

func (c *RedisCache) TTL(key string) (int, error) {
	select {
	case <-c.stop:
		return 0, helper.RedisStorageClose
	default:

	}
	ttl, _ := helper.TTL(c.pool, key)

	return int(ttl), nil

}
