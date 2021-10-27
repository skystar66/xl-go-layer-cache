package entity

type CacheEntity struct {
	Value interface{} `json:"value"`
	//一级缓存过期时间
	FirstExpireTime int `json:"first_expire_time"`
	//二级缓存过期时间
	SencondExpireTime int `json:"sencond_expire_time"`
}

func NewEntity(v interface{}, firstExpireTime, sencondExpireTime int) *CacheEntity {

	//todo 后期二级缓存过期时间可以自己灵活配置
	return &CacheEntity{
		Value:             v,
		FirstExpireTime:   firstExpireTime,
		SencondExpireTime: sencondExpireTime,
	}
}
