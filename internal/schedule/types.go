package schedule

import (
	"github.com/alehua/cron-center/internal/executor"
	"github.com/alehua/cron-center/internal/storage"
	"github.com/alehua/cron-center/internal/task"
	"github.com/alehua/ekit/queue"
	"log"
	"sync"
	"time"
)

type Scheduler struct {
	s          storage.Storager
	tasks      map[string]scheduledTask
	executors  map[executor.ExecType]executor.Executor
	mux        sync.Mutex
	readyTasks *queue.DelayQueue[execution]
	taskEvents chan task.Event
}

type scheduledTask struct {
	task       *task.Task
	executeId  int64
	executor   executor.Executor
	expr       string
	stopped    bool
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

func (r *scheduledTask) run() {
	// 如果这个任务已经被停止/取消了，什么也不做
	if r.stopped {
		return
	}
	select {
	case r.taskEvents <- task.Event{Task: *r.task, Type: task.EventTypeRunning}:
		log.Printf("task id: %d, is running", r.task.TaskId)
	}
	// 这里executor返回一个task.Event,表示任务的执行状态
	taskEvent := r.executor.Execute(r.task)
	select {
	case e := <-taskEvent:
		r.taskEvents <- e
		switch e.Type {
		case task.EventTypeFailed, task.EventTypeSuccess:
			return
		}
	}
}
