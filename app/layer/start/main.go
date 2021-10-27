package main

import (
	"g2cache/app/layer/cache"
	"github.com/gogf/gf/container/gtype"
)

func main() {
	//创建一个缓存

	cache, _ := cache.NewCache(10*1000, 20*1000, 10*60*1000, false)

	cache.Set("go-test1", "nihao")

	cache.Get("go-test1",gtype.String{}, func(key string) (interface{}, error) {


		return nil,nil
	})

	select {
	}

}
