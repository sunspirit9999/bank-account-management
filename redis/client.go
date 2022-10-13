package redis_client

import "github.com/go-redis/redis"

func InitRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
