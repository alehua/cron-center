package schedule

import (
	"context"
	"github.com/alehua/cron-center/internal/executor"
	"github.com/alehua/cron-center/internal/storage"
	"github.com/alehua/cron-center/internal/task"
	"github.com/alehua/ekit/queue"
	"log"
	"sync"
	"time"
)

func NewScheduler(s storage.Storager) *Scheduler {
	sche := &Scheduler{
		storage:    s,
		tasks:      make(map[string]scheduledTask),
		executors:  make(map[string]executor.Executor),
		mux:        sync.Mutex{},
		readyTasks: queue.NewDelayQueue[execution](),
		taskEvents: make(chan task.Event),
	}

	sche.executors = map[string]executor.Executor{}

	return sche
}

func (sche *Scheduler) Start(ctx context.Context) error {
	go func() {
		// 这里进行已经写入延迟队列中的事件执行，并且写入时间执行的结果
		e := sche.executeLoop(ctx)
		if e != nil {
			log.Println(e)
		}
	}()

	events := sche.storage.Events(ctx, sche.taskEvents)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-events:
			switch event.Type {
			case storage.EventTypePreempted:
				// t := sche.newRunningTask(ctx, event.Task, sc.executors[string(event.Task.Type)])
				if exec, exist := sche.executors[event.Task.Type]; !exist {
					log.Println("任务执行类型不存在")
				} else {
					st := scheduledTask{
						task:     event.Task,
						executor: exec,
						// expr:       time.Since(event.Task.Next(time.Now())),
						taskEvents: make(chan task.Event),
					}
					sche.mux.Lock()
					sche.readyTasks.Add(execution{
						scheduledTask: &st,
						time:          event.Task.Next(time.Now()),
					})
				}
			default:
				// 其他类型暂时不支持
			}
		}
	}
}

func (sche *Scheduler) executeLoop(ctx context.Context) error {
	for {
		t, _ := sche.readyTasks.Take() // 如果没有任务这里会阻塞
		go func(exec execution) {
			delay := exec.task.MaxTime
			execCtx, cancel := context.WithTimeout(context.Background(), delay)
			defer cancel()
			event := task.Event{}
			err := exec.executor.Exec(execCtx, *exec.task)
			if err != nil {
				event.Type = task.EventTypeFailed
			} else {
				event.Type = task.EventTypeSuccess
			}
			sche.taskEvents <- event
		}(t)
	}
}
