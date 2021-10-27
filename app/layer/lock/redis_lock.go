package lock

import (
	"g2cache/app/layer/cache/sencond"
	"github.com/gogf/gf/os/glog"
	"sync"
	"time"
)

type RedisLock struct {
	redis *sencond.RedisCache
}

const (
	SCRIPT_LOCK = ` 
    local res=redis.call('GET', KEYS[1])
    if res then
        return 0
    else
        redis.call('SET',KEYS[1],ARGV[1]);
        redis.call('EXPIRE',KEYS[1],ARGV[2])
        return 1
    end 
    `
	SCRIPT_EXPIRE = ` 
    local res=redis.call('GET', KEYS[1])
    if not res then
        return -1
    end 
    if res==ARGV[1] then
        redis.call('EXPIRE', KEYS[1], ARGV[2])
        return 1
    else
        return 0
    end 
    `

	SCRIPT_UN_LOCK = ` 
    local res=redis.call('GET', KEYS[1])
    if not res then 
        return -1
    end 
    if res==ARGV[1] then
        redis.call('DEL', KEYS[1])
    else
        return 0
    end 
    `
)

var (
	onceinstance *RedisLock
	once         sync.Once
)

func NewRedisLock(cache *sencond.RedisCache) *RedisLock {
	once.Do(func() {
		redislock := RedisLock{redis: cache}
		onceinstance = &redislock
	})
	return onceinstance
}

//锁续期
func (l *RedisLock) ResetExpireTime(key string, ttlSencond int) {
	for true {
		if ok, err := l.redis.RenewalExpiretime(key, SCRIPT_EXPIRE, ttlSencond); err != nil {
			glog.Infof("luaExpire exec error", err)
			break
			if !ok {
				glog.Infof("Reset expire failed. key=%s", key)
				break
			}
			glog.Infof("Reset expire succeed. key=%s", key)
		}
		//每五秒检查一次
		time.Sleep(5 * time.Second)
	}
}

//获取锁
func (l *RedisLock) Lock(key string, ttlSencond int) bool {
	ok, _ := l.redis.Lock(key, SCRIPT_LOCK, ttlSencond, 1)
	return ok
}

//释放锁
func (l *RedisLock) UnLock(key string) bool {
	ok, _ := l.redis.Unlock(key, SCRIPT_UN_LOCK)
	return ok
}
