package main

import (
	"fmt"
	"g2cache/app/layer/cache"
	"github.com/gogf/gf/container/gtype"
	"github.com/gogf/gf/util/gconv"
	"github.com/shirou/gopsutil/cpu"
	"sync"
	"time"
)

func main() {
	//创建一个缓存 一级缓存过期时间：20s，二级缓存过期时间：10分钟，预刷新缓存的时间：10s
	cache, err := cache.NewCache(int(10*time.Second), int(20*time.Second), int(2*60*60*time.Second), false)
	fmt.Println(err)
	//
	////
	//cache.Set("test-1","test1")
	//
	//time.Sleep(2*time.Second)
	//
	//
	//cache.Set("test-2","test2")
	//
	//time.Sleep(2*time.Second)
	//
	//
	//cache.Set("test-3","test3")
	//cache
	//压测写入
	AbTestWrite(cache)

	select {}

}

func AbTestWrite(cache *cache.LayeringCache) {

	threadNum := 50   // 线程数
	dataNum := 20000 //每个线程处理的数量

	prefixKey:="abTest:" //key的前缀

	//模拟100万写入的qps，10线程
	start := time.Now().Unix()
	wg := sync.WaitGroup{}
	wg.Add(threadNum)
	for i := 0; i < threadNum; i++ {
		go func(index int) {
			for j := 0; j < dataNum; j++ {
				key := fmt.Sprint(prefixKey+gconv.String(i) + "-" + gconv.String(j))
				cache.Set(key, key)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	end := time.Now().Unix() - start
	if end > 0 {
		totalReq := string(threadNum * dataNum)
		qps := string(int64(threadNum * dataNum*1000) / end)
		cpu:=string(CpuCount())
		fmt.Println( cpu+ "-core->" + totalReq + "请求：use" +
			"：" + string(end) + "ms" +"线程数："+string(threadNum)+ ",qps:" + qps + "/s")
	}
}

func AbTestRead(cache *cache.LayeringCache) {

	threadNum := 50   // 线程数
	dataNum := 20000 //每个线程处理的数量
	prefixKey:="abTest:" //key的前缀
	//模拟100万的qps，10线程
	start := time.Now().Unix()
	wg := sync.WaitGroup{}
	wg.Add(threadNum)
	for i := 0; i < threadNum; i++ {
		go func(index int) {
			for j := 0; j < dataNum; j++ {
				key := fmt.Sprint(prefixKey+gconv.String(i) + "-" + gconv.String(j))
				cache.Get(key, gtype.String{},nil)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	end := time.Now().Unix() - start
	if end > 0 {
		totalReq := string(threadNum * dataNum)
		qps := string(int64(threadNum * dataNum*1000) / end)
		cpu:=string(CpuCount())
		fmt.Println( cpu+ "-core->" + totalReq + "请求：use" +
			"：" + string(end) + "ms" +"线程数："+string(threadNum)+ ",qps:" + qps + "/s")
	}
}



func CpuCount() int {
	num, _ := cpu.Counts(false)
	return num
}
