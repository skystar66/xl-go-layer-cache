package pullmsg

import (
	"fmt"
	"g2cache/app/layer/cache/sencond"
	"g2cache/app/layer/helper"
	sync2 "g2cache/app/layer/sync"
	"github.com/gogf/gf/os/glog"
	"sync"
	"sync/atomic"
)

type PullMsg struct {
	rediscache *sencond.RedisCache
	syncdata   *sync2.SyncDataService
}

var (
	onceinstance *PullMsg
	once         sync.Once
)

func NewPullMsg() *PullMsg {
	redisCache, _ := sencond.NewRedisCache()
	syncdataService := sync2.NewSyncDataService()
	once.Do(func() {
		cache := PullMsg{
			rediscache: redisCache,
			syncdata:   syncdataService,
		}
		onceinstance = &cache
	})
	return onceinstance
}

//拉取redis数据
func (t *PullMsg) PullMsg() {
	//获取redis的最新的offset
	key := fmt.Sprintf(helper.LayeringMsgQueueKey, "node1")
	num, _ := t.rediscache.Llen(key)
	maxOffset := num - 1
	oldOffset := atomic.SwapInt64(&helper.OFFSET, maxOffset)
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
		if helper.CacheDebug {
			glog.Debugf("【缓存同步】redis 通过PULL方式处理本地缓存，startOffset:【0】,endOffset:【%d】,消息内容：%s", endOffset, data)
		}
		t.syncdata.SyncData(&data)
	}
}
