package schedule

import (
	"context"
	"github.com/alehua/cron-center/internal/executor"
	"github.com/alehua/cron-center/internal/storage"
	"github.com/alehua/cron-center/internal/task"
	"github.com/ecodeclub/ekit/queue"
	"golang.org/x/sync/errgroup"
	"log"
	"sync"
	"time"
)

func NewScheduler(s storage.Storager) *Scheduler {
	sche := &Scheduler{
		storage:    s,
		tasks:      make(map[string]scheduledTask),
		executors:  executor.NewLocalFuncExecutor(),
		mux:        sync.Mutex{},
		readyTasks: queue.NewDelayQueue[execution](10),
		taskEvents: make(chan task.Event),
	}

	return sche
}

func (sche *Scheduler) Start(ctx context.Context) error {
	// 启动抢占和续约
	go sche.storage.Preempt(ctx)
	go sche.storage.AutoRefresh(ctx)
	return sche.start(ctx)
}

func (sche *Scheduler) start(ctx context.Context) error {
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
				st := scheduledTask{
					task:     event.Task,
					executor: sche.executors,
					// expr:       time.Since(event.Task.Next(time.Now())),
					taskEvents: make(chan task.Event),
				}
				// 添加到等待队列
				err := sche.readyTasks.Enqueue(ctx, execution{
					scheduledTask: &st,
					time:          event.Task.Next(time.Now()),
				})
				if err != nil {
					log.Println("插入延迟队列失败, err=", err.Error())
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

func (sche *Scheduler) AddTasks(ctx context.Context,
	task *task.Task,
	fn func(ctx context.Context) error) error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		task.NextTime = task.Next(time.Now())
		// 添加数据库
		return sche.storage.Insert(ctx, task)
	})
	eg.Go(func() error {
		// 添加到 executors
		sche.executors.AddLocalFunc(task.Name, fn)
		return nil
	})
	err := eg.Wait()
	if err != nil {
		return err
	}
	return nil
}
