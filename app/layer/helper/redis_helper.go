package helper

import (
	"encoding/json"
	"github.com/gogf/gf/os/glog"
	"github.com/gogf/gf/util/gconv"
	"github.com/gomodule/redigo/redis"
	"time"
)
//默认的redis配置
var DefaultRedisConf RedisConf

//默认的redis pubs sub配置
var DefaultPubSubRedisConf RedisConf


func InitRedis() {

	DefaultRedisConf.HOST = "127.0.0.1:6379"
	DefaultRedisConf.DB = 1
	DefaultRedisConf.MaxConn = 10
	DefaultPubSubRedisConf = DefaultRedisConf

	glog.Info("redis  初始化成功！！！")

}
type RedisConf struct {
	HOST    string
	Pwd     string
	DB      int
	MaxConn int
}
//redis pub sub channel
func RedisPublish(pool *redis.Pool, channel string, message string) error {
	conn, err := GetRedisConn(pool)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, errPublish := conn.Do("PUBLISH", channel, message)
	return errPublish
}

//redis set key ttl value
func RedisSetEXString(pool *redis.Pool, key, value string, ttl int64) error {
	conn, err := GetRedisConn(pool)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, errSet := conn.Do("SETEX", key, ttl, value)
	return errSet
}

// LLen 获取列表的长度
func LLen(pool *redis.Pool, key string) (int64, error) {
	conn, err := GetRedisConn(pool)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	num, _ := conn.Do("RPOP", key)
	return gconv.Int64(num), err
}

// LRange 返回列表 key 中指定区间内的元素，区间以偏移量 start 和 stop 指定。
// 下标(index)参数 start 和 stop 都以 0 为底，也就是说，以 0 表示列表的第一个元素，以 1 表示列表的第二个元素，以此类推。
// 你也可以使用负数下标，以 -1 表示列表的最后一个元素， -2 表示列表的倒数第二个元素，以此类推。
// 和编程语言区间函数的区别：end 下标也在 LRANGE 命令的取值范围之内(闭区间)。
func LRange(pool *redis.Pool, key string, start, end int64) (interface{}, error) {
	conn, err := GetRedisConn(pool)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return conn.Do("LRANGE", key, start, end)
}

// LPush 将一个值插入到列表头部
func LPush(pool *redis.Pool, key string, member interface{}) error {
	conn, err := GetRedisConn(pool)
	if err != nil {
		return err
	}
	defer conn.Close()
	value, err := encode(member)
	if err != nil {
		return err
	}
	_, err = conn.Do("LPUSH", key, value)
	return err
}

//Get key return value
func RedisGetString(pool *redis.Pool, key string) (string, error) {
	conn, err := GetRedisConn(pool)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	data, errGet := redis.String(conn.Do("GET", key))
	if errGet != nil {
		return "", errGet
	}
	return data, nil
}

//redis del key
func RedisDel(pool *redis.Pool, key string) error {
	conn, err := GetRedisConn(pool)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, errSet := conn.Do("DEL", key)
	return errSet
}

// TTL 以秒为单位。当 key 不存在时，返回 -2 。 当 key 存在但没有设置剩余生存时间时，返回 -1
func TTL(pool *redis.Pool, key string) (ttl int64, err error) {
	conn, err := GetRedisConn(pool)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	ttls, _ := conn.Do("TTL", key)
	return gconv.Int64(ttls), nil
}

// Expire 设置键过期时间，expire的单位为秒
func Expire(pool *redis.Pool, key string, expire int64) error {
	conn, err := GetRedisConn(pool)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.Do("EXPIRE", key, expire)
	return nil
}

//获取Redis连接
func GetRedisConn(pool *redis.Pool) (redis.Conn, error) {
	conn := pool.Get()
	if err := conn.Err(); err != nil {
		return nil, err
	}
	return conn, nil
}

//获取redis连接池
func GetRedisPool() (*redis.Pool, error) {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", DefaultRedisConf.HOST)
			if err != nil {
				return nil, err
			}
			if DefaultRedisConf.Pwd != "" {
				if _, err := c.Do("AUTH", DefaultRedisConf.Pwd); err != nil {
					errC := c.Close()
					if errC != nil {
						return nil, errC
					}
					return nil, err
				}
			}
			if DefaultRedisConf.DB > 0 {
				if _, err := c.Do("SELECT", DefaultRedisConf.DB); err != nil {
					errC := c.Close()
					if errC != nil {
						return nil, errC
					}
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:         DefaultRedisConf.MaxConn,
		MaxActive:       DefaultRedisConf.MaxConn,
		IdleTimeout:     300 * time.Second,
		Wait:            true,
		MaxConnLifetime: 30 * time.Minute,
	}

	//ping
	conn, err := pool.Dial()
	if err != nil {
		return nil, err
	}
	err = pool.TestOnBorrow(conn, time.Now())
	if err != nil {
		return nil, err
	}

	return pool, nil
}

// encode 序列化要保存的值
func encode(val interface{}) (interface{}, error) {
	var value interface{}
	switch v := val.(type) {
	case string, int, uint, int8, int16, int32, int64, float32, float64, bool:
		value = v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		value = string(b)
	}
	return value, nil
}

// decode 反序列化保存的struct对象
func Decode(reply interface{}, err error, val interface{}) error {
	str, err := redis.String(reply, err)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(str), val)
}
