package pubsub

import (
	"g2cache/app/bizseer/cache/first"
	"g2cache/app/bizseer/cache/interface"
	"g2cache/app/bizseer/cache/sencond"
	"g2cache/app/bizseer/helper"
	"g2cache/app/bizseer/pool"
	sync2 "g2cache/app/bizseer/sync"
	"github.com/gogf/gf/os/glog"
	jsoniter "github.com/json-iterator/go"
	"sync"
	"sync/atomic"
	"time"
)

//维护一个offset
var OFFSET int64=0

type PubSubService struct {
	//线程池
	pool     *pool.Pool
	stop     chan struct{}
	stopOnce sync.Once
	//发布订阅消息
	channel chan *_interface.ChannelMetedata
	//redis缓存
	redis *sencond.RedisCache
	//一级缓存
	firstcache *first.FreeCache
	//处理队列消息同步服务
	syncdata *sync2.SyncDataService
}

//初始化
func InitPubSub() {
	//开启订阅
	pubsubs := NewPubSub()
	//redis发起订阅
	pubsubs.Subscribe()
	//接收处理订阅到的消息
	pubsubs.pool.SendJob(pubsubs.subscribeHandler)
	glog.Info("pubsub 初始化 成功！！！")
}

var (
	onceinstance *PubSubService
	once         sync.Once
)

func NewPubSub() *PubSubService {
	pool := pool.NewPool(helper.DefaultGPoolWorkerNum, helper.DefaultGPoolJobQueueChanLen)
	freecache := first.NewFirstCache()
	redisCache, _ := sencond.NewRedisCache()
	syncdataservice:=sync2.NewSyncDataService()

	once.Do(func() {
		pubsub := PubSubService{
			pool:       pool,
			stop:       make(chan struct{}, 1),
			stopOnce:   sync.Once{},
			redis:      redisCache,
			firstcache: freecache,
			channel:    make(chan *_interface.ChannelMetedata, 1024), //默认消息通道大小1024个
			syncdata: syncdataservice,
		}
		onceinstance = &pubsub
	})
	return onceinstance
}
func (s *PubSubService) Subscribe() bool {
	select {
	case <-s.stop:
		return false
	default:

	}
	s.pool.SendJob(func() {
		//订阅redis
		s.redis.Subscribe(s.channel)
	})
	return true
}

//订阅handler【实时同步的】
func (s *PubSubService) subscribeHandler() {
	//pub/sub 数据的处理
	for meta := range s.channel {
		select {
		case <-s.stop:
			return
		default:

		}
		if meta.Key == "" {
			glog.Infof("subscribeHandle receive meta.Key is null: %+v\n", meta)
			continue
		}
		metaDump, _ := jsoniter.MarshalToString(meta)
		glog.Infof("subscribeHandle receive meta: %v\n", metaDump)
		//消息的偏移量+1
		atomic.AddInt64(&OFFSET, 1)
		//设置最后一次的推送时间
		atomic.SwapInt64(&helper.LAST_PUSH_TIME, time.Now().Unix())
		//处理业务数据
		s.syncdata.SyncData(meta)
	}
}
