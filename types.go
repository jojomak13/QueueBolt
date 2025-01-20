package bolt

import "github.com/go-redis/redis/v8"

type Job struct {
	ID        string
	Payload   string
	ExecuteAt int64
}

type Bolt struct {
	client       *redis.Client
	workersCount int
}
