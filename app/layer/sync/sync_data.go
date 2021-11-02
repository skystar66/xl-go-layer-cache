package sync

import (
	"g2cache/app/layer/cache/first"
	"g2cache/app/layer/cache/interface"
	"g2cache/app/layer/cache/sencond"
	"g2cache/app/layer/helper"
	"github.com/gogf/gf/os/glog"
	jsoniter "github.com/json-iterator/go"
	"sync"
	"time"
)

type SyncDataService struct {
	//redis缓存
	redis *sencond.RedisCache
	//一级缓存
	firstcache *first.FreeCache
}
var (
	onceinstance *SyncDataService
	once         sync.Once
)
func NewSyncDataService() *SyncDataService{
	freecache := first.NewFirstCache()
	redisCache, _ := sencond.NewRedisCache()
	once.Do(func() {
		cache := SyncDataService{
			firstcache: freecache,
			redis: redisCache,
		}
		onceinstance = &cache
	})
	return onceinstance
}

//处理业务数据
func (s *SyncDataService) SyncData(metedata *_interface.ChannelMetedata) {
	switch metedata.Action {
	case helper.SetPublishType:
		/**更新一级缓存*/
		s.firstcache.SetExpireTime(metedata.Key, metedata.Data, time.Duration(metedata.Data.FirstExpireTime))
		value,_:=jsoniter.MarshalToString(metedata.Data)
		if helper.CacheDebug {
			glog.Debugf("【一级缓存同步】更新一级缓存数据,key=%s,消息内容=%s", metedata.Key,value )
		}
	case helper.DelPublishType:
		/**清除一级缓存*/
		s.firstcache.Evict(metedata.Key)
		if helper.CacheDebug {
			glog.Debugf("【一级缓存同步】删除一级缓存数据,key=%s", metedata.Key)
		}
	}
}
