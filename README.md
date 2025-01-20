## QueueBolt

```golang
package main

import (
	"time"

	"github.com/go-redis/redis/v8"
	bolt "github.com/jojomak13/QueueBolt"
)

func main() {
	queue := bolt.NewQueue(redis.Options{
		Addr: "localhost:6379",
	}, 1)

	queue.Run()

	queue.Add(bolt.Job{
		ID:        "3248932489",
		Payload:   "welcome jojo",
		ExecuteAt: time.Now().Add(time.Second * 2).Unix(),
	})

	queue.Add(bolt.Job{
		ID:        "32489",
		Payload:   "welcome jojo again",
		ExecuteAt: time.Now().Add(time.Second * 4).Unix(),
	})

	select {
	//
	}
}
```
