package main

import (
	"context"
	"github.com/alehua/cron-center/internal/ioc"
	"github.com/alehua/cron-center/internal/ioc/demo"
	"github.com/alehua/cron-center/internal/task"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sche := ioc.Schedule()
	err := sche.AddTasks(ctx, &task.Task{
		Config: task.Config{
			Name:    "demo",
			Cron:    "0 0/2 * * * ?",
			MaxTime: 10 * time.Second,
		},
	}, demo.Demo)
	if err != nil {
		panic(err)
	}
	if err := sche.Start(ctx); err != nil {
		panic(err)
	}
}
