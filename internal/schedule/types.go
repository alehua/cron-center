package schedule

import (
	"github.com/alehua/cron-center/internal/executor"
	"github.com/alehua/cron-center/internal/storage"
	"github.com/alehua/cron-center/internal/task"
	"github.com/ecodeclub/ekit/queue"
	"sync"
	"time"
)

type Scheduler struct {
	storage    storage.Storager
	tasks      map[string]scheduledTask
	executors  map[string]executor.Executor
	mux        sync.Mutex
	readyTasks *queue.DelayQueue[execution]
	taskEvents chan task.Event
}

type scheduledTask struct {
	task     *task.Task
	executor executor.Executor
	// expr       time.Duration
	taskEvents chan task.Event
}

type execution struct {
	*scheduledTask
	time time.Time
}

func (e execution) Delay() time.Duration {
	//return e.time.Sub(time.Now())
	return time.Until(e.time)
}
