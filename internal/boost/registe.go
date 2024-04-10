package boost

import (
	"context"
	"github.com/alehua/cron-center/internal/schedule"
	"github.com/alehua/cron-center/internal/task"
	"time"
)

func register(ctx context.Context, sche *schedule.Scheduler) int {
	task1 := &task.Task{
		Config: task.Config{
			Name:    "demo",
			Cron:    "0 0/2 * * * ?",
			Type:    "local",
			MaxTime: 10 * time.Second,
		},
	}

	err := sche.AddTasks(ctx, task1)
	if err != nil {
		return 0
	}
	return 1
}
