package bolt

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

func NewQueue(redisConfig redis.Options, workersCount int) *Bolt {
	client := redis.NewClient(&redisConfig)

	return &Bolt{client: client, workersCount: workersCount}
}

func (jq *Bolt) Add(job Job) error {
	ctx := context.Background()

	err := jq.client.ZAdd(ctx, "delayed_jobs", &redis.Z{
		Score:  float64(job.ExecuteAt),
		Member: job.ID,
	}).Err()

	if err == nil {
		err = jq.client.HSet(ctx, "job:"+job.ID, "payload", job.Payload).Err()
	}

	return err
}

func (jq *Bolt) tryAcquireLock(ctx context.Context, jobID string) bool {
	// Try to set the lock key with an expiration of 30 seconds
	lockKey := fmt.Sprintf("lock:%s", jobID)
	ok, err := jq.client.SetNX(ctx, lockKey, "1", 30*time.Second).Result()
	if err != nil {
		return false
	}
	return ok
}

func (jq *Bolt) releaseLock(ctx context.Context, jobID string) {
	lockKey := fmt.Sprintf("lock:%s", jobID)
	jq.client.Del(ctx, lockKey)
}

func (jq *Bolt) getReadyJobs() ([]Job, error) {
	ctx := context.Background()

	readyJobs, err := jq.client.ZRangeByScore(ctx, "delayed_jobs", &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", time.Now().Unix()),
	}).Result()

	if err != nil {
		return nil, err
	}

	var processedJobs []Job

	for _, jobID := range readyJobs {
		if !jq.tryAcquireLock(ctx, jobID) {
			continue
		}

		payload, err := jq.client.HGet(ctx, "job:"+jobID, "payload").Result()
		if err != nil {
			jq.releaseLock(ctx, jobID)
			continue
		}

		processedJobs = append(processedJobs, Job{
			ID:      jobID,
			Payload: payload,
		})
	}

	return processedJobs, nil
}

func (jq *Bolt) Acknowledge(jobID string) error {
	ctx := context.Background()

	pipe := jq.client.Pipeline()
	pipe.ZRem(ctx, "delayed_jobs", jobID)
	pipe.Del(ctx, "job:"+jobID)
	_, err := pipe.Exec(ctx)

	jq.releaseLock(ctx, jobID)

	return err
}

func (jq *Bolt) process() {
	for {
		readyJobs, err := jq.getReadyJobs()
		if err != nil {
			fmt.Println("Error processing jobs:", err)
		}

		for _, job := range readyJobs {
			fmt.Printf("Processing job: %s\n", job.Payload)
			time.Sleep(time.Second * 2)
			jq.Acknowledge(job.ID)
		}

		// Wait before next check
		time.Sleep(1 * time.Second)
	}
}

func (jq *Bolt) Run() {
	for i := 0; i < jq.workersCount; i++ {
		go jq.process()
	}
}
