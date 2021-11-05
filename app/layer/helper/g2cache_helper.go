package helper

import (
	"errors"
	"fmt"
	"github.com/mohae/deepcopy"
	"reflect"
)

var (

	//==================================错误提示配置==================================

	//本地缓存关闭
	LocalStorageClose = errors.New("local storage close !!! ")
	//redis缓存关闭
	RedisStorageClose = errors.New("redis storage close !!! ")
	CacheClose        = errors.New("layer cache close !!! ")
	KeyNotEmpty       = errors.New("key not empty !!!")
	ObjNotEmpty       = errors.New("cache obj not empty !!!")
	LoadDataNotEmpty  = errors.New("load  data function not empty !!!")
	//==================================redis配置==================================
	Redis_HOST    = "127.0.0.1:6379"
	Redis_PWD     = ""
	Redis_DB      = 0
	Redis_MaxConn = 50
	//==================================线程队列配置==================================

	DefaultGPoolWorkerNum       = 100    // 线程数量
	DefaultGPoolJobQueueChanLen = 400000 //任务队列

	//==================================周期性任务配置==================================

	PULL_MSG_TIME_SENCOND      = 30 //默认30s执行一次拉取消息的任务
	CacheMonitorSecond         = 5  //缓存监控
	CacheMonitorJobQueueSecond = 5  //任务队列监控

	//==================================日志开关配置==================================
	CacheDebug = false

	//==================================redis数据同步配置==================================
	LAST_PUSH_TIME int64 = 0 //最后一次处理推消息的时间戳
	OFFSET         int64 = 0 //维护拉取消息队列的offset
)

const (
	LayeringTermExpireTime   = "layering-cache:term-redis:%s"
	LayeringExecuteDBTime    = "layering-cache:execute-db:%s"
	Cron_Clean_Message_Queue = "0 0 3 * * ?" //每天凌晨3点执行一次

	//缓存主动在失效前强制刷新缓存的时间，默认20s
	PreloadTime       int64 = 20
	RECONNECTION_TIME int64 = 10 * 1000 //pub/sub 重连时间间隔

	LayeringMsgQueueKey string = "layering-cache:message-key:%s"

	Lock_Prefix = "layer_cache_lock_prefix_"

	//==================================数据同步类型==================================

	//更新、添加
	SetPublishType int64 = iota
	//清除缓存
	DelPublishType
)

func Clone(src, dst interface{}) (err error) {

	defer func() {
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprint(e))
			return
		}
	}()

	v := deepcopy.Copy(src)
	if reflect.ValueOf(v).IsValid() {
		reflect.ValueOf(dst).Elem().Set(reflect.Indirect(reflect.ValueOf(v)))
	}
	return err
}

//默认值
type NullValue struct {
}
