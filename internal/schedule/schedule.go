package schedule

import (
	"context"
	"github.com/alehua/cron-center/internal/executor"
	"github.com/alehua/cron-center/internal/executor/local"
	"github.com/alehua/cron-center/internal/storage"
	"github.com/alehua/cron-center/internal/task"
	"github.com/ecodeclub/ekit/queue"
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
		readyTasks: queue.NewDelayQueue[execution](10),
		taskEvents: make(chan task.Event),
	}

	sche.executors = map[string]executor.Executor{
		"local": local.NewLocalFuncExecutor(),
	}

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
					err := sche.readyTasks.Enqueue(ctx, execution{
						scheduledTask: &st,
						time:          event.Task.Next(time.Now()),
					})
					if err != nil {
						log.Println("插入延迟队列失败, err=", err.Error())
					}
				}
			default:
				// 其他类型暂时不支持
			}
		}
	}
}

func (sche *Scheduler) executeLoop(ctx context.Context) error {
	for {
		t, err := sche.readyTasks.Dequeue(ctx) // 如果没有任务这里会阻塞
		if err != nil {
			log.Println("获取任务失败, err=", err.Error())
		}
		select {
		// 检测 ctx 有没有过期
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
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

func (sche *Scheduler) AddTasks(ctx context.Context, tasks ...*task.Task) error {
	for _, t := range tasks {
		t.NextTime = t.Next(time.Now())
		err := sche.storage.Insert(ctx, t)
		if err != nil {
			return err
		}
	}
	return nil
}
