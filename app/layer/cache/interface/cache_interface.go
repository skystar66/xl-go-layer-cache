package _interface

import (
	"g2cache/app/layer/entity"
	"time"
)

//一级缓存
type FirstCache interface {
	/**获取缓存结果*/
	Get(key string, resultObj interface{}) (*entity.CacheEntity, bool, error)
	//设置key value 缓存
	Set(key string, e *entity.CacheEntity) error
	//设置key value 缓存带有过期时间的
	SetExpireTime(key string, e *entity.CacheEntity, duration time.Duration) error
	//删除缓存中对应的key
	Evict(key string) error
	//清除缓存
	Clear()
	//是否是线程安全的
	ThreadSafe() // Need to ensure thread safety
	//关闭缓存
	Close()
}

//二级缓存
type SencondCache interface {
	/**获取缓存结果*/
	Get(key string, resultObj interface{}) (*entity.CacheEntity, bool, error)
	//设置key value 缓存
	Set(key string, e *entity.CacheEntity) error
	//设置key value 缓存带有过期时间的
	SetExpireTime(key string, e *entity.CacheEntity, duration time.Duration) error
	ExpireTime(key string,duration time.Duration) error
	//删除缓存中对应的key
	Evict(key string) error
	//清除缓存
	Clear()
	//是否是线程安全的
	ThreadSafe() // Need to ensure thread safety
	//关闭缓存
	Close()
	//获取消息队列长度
	Llen(key string) (int64, error)
	//获取list的范围数据
	LRange(key string, startOffset, endOffset int64) ([]*ChannelMetedata, error)
	// LPush 将一个值插入到列表头部
	LPush(key string, member interface{}) error
	TTL(key string) (int,error)
	Lock(key,script string, ttlSencond,kecount int) (bool, error)
	Unlock(key,script string) (bool, error)
	RenewalExpiretime(key,script string, ttlSencond int) (bool, error)
}

//sencond pub sub订阅接口
type PubSub interface {
	Subscribe(data chan<- *ChannelMetedata) error
	Publish(data *ChannelMetedata) error
}

//缓存不存在时，提供一个load的函数，进行获取数据库
type LoadDataSourceFunc func(key string) (interface{}, error)

//传输数据通道
type ChannelMetedata struct {
	Key string `json:"key"`
	//工作组id
	Gid string `json:"gid"`
	//操作： SetPublishType,DelPublishType
	Action int64 `json:"action"`
	//通道内的数据
	Data *entity.CacheEntity
}
