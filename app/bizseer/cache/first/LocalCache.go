package first

import (
	"encoding/json"
	"g2cache/app/bizseer/entity"
	"g2cache/app/bizseer/helper"
	"github.com/coocood/freecache"
	"sync"
	"time"
)

var (
	DefaultFreeCacheSize = 50 * 1024 * 1024 // 默认本地缓存大小50MB

)
type FreeCache struct {
	storage  *freecache.Cache
	stop     chan struct{} //缓存关闭 会触发stop
	stopOnce sync.Once     //线程安全的缓存关闭
}

var (
	onceFirstCache *FreeCache
	onceFirst         sync.Once
)
//创建一个一级缓存
func NewFirstCache() *FreeCache {
	onceFirst.Do(func() {
		cache := FreeCache{
			storage: freecache.NewCache(DefaultFreeCacheSize),
			stop:    make(chan struct{}, 1),
		}
		onceFirstCache = &cache
	})
	return onceFirstCache
}


func (f *FreeCache) Set(key string, e *entity.CacheEntity) error {
	//校验本地缓存是否关闭[线程安全]
	select {
	case <-f.stop:
		return helper.LocalStorageClose
	default:

	}
	//序列化
	data, _ := json.Marshal(e)
	//存入本地缓存中
	f.storage.Set([]byte(key), data, -1)
	return nil
}

func (f *FreeCache) SetExpireTime(key string, e *entity.CacheEntity, duration time.Duration) error {

	//校验本地缓存是否关闭[线程安全]
	select {
	case <-f.stop:
		return helper.LocalStorageClose
	default:

	}
	//序列化
	data, _ := json.Marshal(e)
	//存入本地缓存中 待着过期时间
	f.storage.Set([]byte(key), data, int(duration.Seconds()))
	return nil

}

//根据key 获取value
func (f *FreeCache) Get(key string, resultObj interface{}) (*entity.CacheEntity, bool, error) {
	//校验本地缓存是否关闭[线程安全]
	select {
	case <-f.stop:
		return nil, false, helper.LocalStorageClose
	default:

	}
	//查询本地缓存
	value, err := f.storage.Get([]byte(key))

	if err != nil {
		if err == freecache.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	entity := new(entity.CacheEntity)
	entity.Value = resultObj
	err = json.Unmarshal(value, entity)
	if err != nil {
		return nil, false, err
	}
	return entity, true, nil
}

func (f *FreeCache) Evict(key string) error {

	select {
	case <-f.stop:
		return helper.LocalStorageClose
	default:
	}
	f.storage.Del([]byte(key))
	return nil

}

func (f *FreeCache) close() {
	//关闭缓存
	close(f.stop)
	//清理缓存
	f.storage.Clear()
}

func (f *FreeCache) Close() {
	f.stopOnce.Do(f.close)
}

func (f *FreeCache) ThreadSafe() {}

func (f *FreeCache) Clear() {
	f.storage.Clear()
}
