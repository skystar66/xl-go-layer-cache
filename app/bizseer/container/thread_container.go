package container

import (
	"container/list"
	"github.com/gogf/gf/os/glog"
	"github.com/orcaman/concurrent-map"
	"sync"
	"time"
)

var threadContainerMap = cmap.New()

const (
	defaultShards         = 256
	defaultShardsAndOpVal = 255
)

func InitThreadTimeout() {

	//todo 自动检测超时线程，自动释放
	//CheckTimeoutThread()

	glog.Info("自动检测线程  初始化成功！！！")

}

type Harsher interface {
	Sum64(string) uint64
}

type ThreadContainer struct {
	shards [defaultShards]sync.Mutex
	hash   Harsher
}

type ThreadCondMap struct {
	cond *sync.Cond
	//存入时间
	time int64
	//超时时间
	timeout int64
}

var (
	onceinstance *ThreadContainer
	once         sync.Once
)

func NewThreadContainer() *ThreadContainer {
	once.Do(func() {
		tc := ThreadContainer{
			hash: new(fnv64a),
		}
		onceinstance = &tc
	})
	return onceinstance
}

func (t *ThreadContainer) Await(key string, cond *sync.Cond,timeout int64) {
	t.getShardsLock(key)
	tc, _ := threadContainerMap.Get(key)
	var lists *list.List
	if tc == nil {
		lists = list.New()
	} else {
		lists = tc.(*list.List)
	}
	tcm := ThreadCondMap{
		cond: cond,
		time: time.Now().Unix(),
		timeout: timeout,
	}
	lists.PushBack(tcm)
	t.releaseShardsLock(key)
	threadContainerMap.Set(key, lists)
	cond.Wait()
}

func (t *ThreadContainer) NotifyAll(key string) {
	tc, _ := threadContainerMap.Get(key)
	if tc != nil {
		lists := tc.(*list.List)
		for i := lists.Front(); i != nil; i = i.Next() {
			cond := i.Value.(*ThreadCondMap)
			cond.cond.Signal()
		}
	}
}

//自动检测超时线程，自动释放
func CheckTimeoutThread() {
	go func() {
		for true {
			for _, value := range threadContainerMap.Items() {
				lists := value.(*list.List)
				for i := lists.Front(); i != nil; i = i.Next() {
					tcm := i.Value.(*ThreadCondMap)
					//校验锁是否超时
					if (time.Now().Unix() - tcm.time) >= tcm.timeout {
						//大于500ms自动释放
						tcm.cond.Signal()
					}
				}
			}
			//每秒执行一遍
			time.Sleep(1*time.Second)
		}
	}()
}

// This function may block
func (t *ThreadContainer) getShardsLock(key string) {
	idx := t.hash.Sum64(key)
	t.shards[idx&defaultShardsAndOpVal].Lock()
}

func (t *ThreadContainer) releaseShardsLock(key string) {
	idx := t.hash.Sum64(key)
	t.shards[idx&defaultShardsAndOpVal].Unlock()
}

const (
	// offset64 FNVa offset basis. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	offset64 = 14695981039346656037
	// prime64 FNVa prime value. See https://en.wikipedia.org/wiki/Fowler–Noll–Vo_hash_function#FNV-1a_hash
	prime64 = 1099511628211
)

type fnv64a struct{}

// Sum64 gets the string and returns its uint64 hash value.
func (f fnv64a) Sum64(key string) uint64 {
	var hash uint64 = offset64
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}

	return hash
}
