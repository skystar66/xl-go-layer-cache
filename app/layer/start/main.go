package main

import (
	"fmt"
	"g2cache/app/layer/cache"
	"g2cache/app/layer/helper"
	"github.com/gogf/gf/container/gtype"
	"github.com/gogf/gf/os/glog"
	"github.com/gogf/gf/util/gconv"
	"github.com/shirou/gopsutil/cpu"
	"sync"
	"time"
)

func main() {

	//glog.Path("/Users/xuliang/Documents/g2cache/log/g2cache.log")
	glog.SetAsync(true)
	glog.SetStdoutPrint(false)

	glog.SetDebug(false)
	helper.CacheDebug = false
	glog.SetPath("/Users/xuliang/Documents/g2cache/log/g2cache.log")

	//创建一个缓存 一级缓存过期时间：20s，二级缓存过期时间：6小时，预刷新缓存的时间：10s
	cache, _ := cache.NewCache(int(10*time.Second), int(50*time.Second), int(6*60*60*time.Second), false)

	//todo 测试一下超时自动恢复

	//TestCache(cache)
	//cache
	//压测写入
	//for i := 0; i < 2; i++ {
	//	AbTestWrite(cache)
	//}

	//压测读取
	for i := 0; i < 10; i++ {
		AbTestRead(cache)
	}

	//CpuCount()
	select {}

}

func TestCache(cache *cache.LayeringCache) {

	key1 := "qqq"
	key2 := "wwww"
	cache.Set(key1,key1)
	cache.Set(key2,key2)

	cache.Delete(key1)
	cache.Delete(key2)
	//for g := 0; g < 1; g++ {
	//	go func(index int) {
	//		value, _ := cache.Get(key1, gtype.String{}, func(key string) (interface{}, error) {
	//			glog.Debugf("key=%s,获取数据库数据！！！", key)
	//			return key, nil
	//		})
	//		fmt.Printf(
	//			"key=%s,获取到value:%s \n", key1, gconv.String(value))
	//	}(g)
	//	//go func(index int) {
	//	//	value,_:=cache.Get(key2,gtype.String{}, func(key string) (interface{}, error) {
	//	//		glog.Debugf("key=%s,获取数据库数据！！！",key)
	//	//		return key,nil
	//	//	})
	//	//	fmt.Printf(
	//	//		"key=%s,获取到value:%s \n",key2,gconv.String(value))
	//	//}(g)
	//}

	//cache.Set("test-1","test1")
	//
	//time.Sleep(2*time.Second)
	//val,_:=cache.Get("test-1",gtype.String{}, func(key string) (interface{}, error) {
	//	return key,nil
	//})
	//
	//fmt.Println(val)
	//
	//cache.Set("test-2","test2")
	//
	//time.Sleep(2*time.Second)
	//
	//
	//cache.Set("test-3","test3")
}

func AbTestWrite(cache *cache.LayeringCache) {

	threadNum := 150               // 线程数
	dataNum := 1000000 / threadNum //每个线程处理的数量

	prefixKey := "abTest:" //key的前缀

	//模拟100万写入的qps，10线程
	start := time.Now().Unix()
	wg := sync.WaitGroup{}
	wg.Add(threadNum)
	for i := 0; i < threadNum; i++ {
		go func(index int) {
			for j := 0; j < dataNum; j++ {
				key := fmt.Sprint(prefixKey + gconv.String(index) + "-" + gconv.String(j))
				cache.Set(key, key)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	end := time.Now().Unix() - start
	if end > 0 {
		totalReq := gconv.String(threadNum * dataNum)
		qps := gconv.String((int64(threadNum*dataNum) / end) * 20)
		cpu := gconv.String(CpuCount())
		fmt.Println(time.Now().String()+","+cpu + "-core->" + totalReq + "请求：use" +
			"：" + gconv.String(end) + "s" + ",线程数：" + gconv.String(threadNum) + ",qps:" + qps + "/s")
	}
}

//压测读取纯内存+redis+数据库
func AbTestRead(cache *cache.LayeringCache) {

	threadNum := 200               // 线程数
	dataNum := 1000000 / threadNum //每个线程处理的数量
	prefixKey := "abTest:"         //key的前缀
	//模拟100万的qps，10线程
	start := time.Now().Unix()
	wg := sync.WaitGroup{}
	wg.Add(threadNum)
	for i := 0; i < threadNum; i++ {
		go func(index int) {
			for j := 0; j < dataNum; j++ {
				key := fmt.Sprint(prefixKey + gconv.String(index) + "-" + gconv.String(j))
				cache.Get(key, gtype.String{}, func(key string) (interface{}, error) {
					if helper.CacheDebug {
						glog.Debugf("key=%s 获取数据库数据！! ", key)
					}
					return key, nil
				})
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	end := time.Now().Unix() - start
	if end > 0 {
		totalReq := gconv.String(threadNum * dataNum)
		qps := gconv.String(int64(threadNum*dataNum) / end)
		cpu := gconv.String(CpuCount())
		fmt.Println(time.Now().String()+","+cpu + "-core->" + totalReq + "请求：use" +
			"：" + gconv.String(end) + "s" + ",线程数：" + gconv.String(threadNum) + ",qps:" + qps + "/s")
	}
}

func CpuCount() int {
	num, _ := cpu.Counts(false)
	return num
}
