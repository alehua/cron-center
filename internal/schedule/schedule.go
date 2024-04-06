package schedule

import (
	"context"
	"github.com/alehua/cron-center/internal/executor"
	"github.com/alehua/cron-center/internal/storage"
	"github.com/alehua/cron-center/internal/task"
	"github.com/alehua/ekit/queue"
	"log"
	"sync"
)

func NewScheduler(s storage.Storager) *Scheduler {
	sche := &Scheduler{
		s:          s,
		tasks:      make(map[string]scheduledTask),
		executors:  make(map[executor.ExecType]executor.Executor),
		mux:        sync.Mutex{},
		readyTasks: queue.NewDelayQueue[execution](),
		taskEvents: make(chan task.Event),
	}

	sche.executors = map[executor.ExecType]executor.Executor{}

	return sche
}

func (sc *Scheduler) Start(ctx context.Context) error {
	go func() {
		// 这里进行已经写入延迟队列中的事件执行，并且写入时间执行的结果
		e := sc.executeLoop(ctx)
		if e != nil {
			log.Println(e)
		}
	}()

	events, err := sc.s.Events(ctx, sc.taskEvents)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-events:
			switch event.Type {
			case storage.EventTypePreempted:
				t := sc.newRunningTask(ctx, event.Task, sc.executors[string(event.Task.Type)])
				sc.mux.Lock()
				sc.tasks[t.task.Name] = t
				sc.mux.Unlock()
				_ = sc.readyTasks.Enqueue(ctx, execution{
					scheduledTask: &t,
					// TODO 当前demo只是跑的一次性任务，需要适配定时任务
					time: t.next(),
				})
				log.Println("preempted success, enqueued done")
			case storage.EventTypeDeleted:
				sc.mux.Lock()
				tn, ok := sc.tasks[event.Task.Name]
				delete(sc.executors, event.Task.Name)
				sc.mux.Unlock()
				if ok {
					tn.stop()
				}
			}
		}
	}
}

func (sc *Scheduler) executeLoop(ctx context.Context) error {
	for {
		t, _ := sc.readyTasks.Take()
		go t.
		for {
			select {
			case te := <-t.taskEvents:
				switch te.Type {
				case task.EventTypeRunning:
					if er := sc.s.CompareAndUpdateTaskExecutionStatus(ctx, t.task.TaskId,
						task.EventTypeInit, task.EventTypeRunning); err != nil {
						log.Println("sche running: ", er)
					}
					log.Println("scheduler 收到task running信号")
				case task.EventTypeSuccess:
					if er := sc.s.CompareAndUpdateTaskExecutionStatus(ctx, t.task.TaskId,
						task.EventTypeRunning, task.EventTypeSuccess); err != nil {
						log.Println("sche succ: ", er)
					}
					log.Println("scheduler 收到task run success信号:", t.task.TaskId)
				case task.EventTypeFailed:
					if er := sc.s.CompareAndUpdateTaskExecutionStatus(ctx, t.task.TaskId,
						task.EventTypeRunning, task.EventTypeFailed); err != nil {
						log.Println("sche fail: ", er)
					}
					log.Println("scheduler 收到task run fail信号")
				}
				sc.taskEvents <- te
			}
		}
	}
}

// Start 开始调度。当被取消，或者超时的时候，就会结束调度
//func (s *Scheduler) Start(ctx context.Context) error {
//	// 启动任务执行
//	go s.execute(ctx)
//	// 抢占任务间隔
//	tickerP := time.NewTicker(s.preemptInterval)
//	defer tickerP.Stop()
//	for {
//		if ctx.Err() != nil {
//			return ctx.Err()
//		}
//		select {
//		case <-tickerP.C:
//			log.Printf("cron-center: scheduler[%d] 开始进行任务抢占", s.executeId)
//			go s.preempted()
//		case <-s.stop:
//			log.Printf("cron-center: scheduler[%d] 停止任务抢占", s.executeId)
//			return nil
//		default:
//		}
//	}
//}

//func (s *Scheduler) execute(ctx context.Context) {
//	for {
//		select {
//		case <-ctx.Done():
//			log.Printf("cron-center: scheduler[%d] 停止任务执行", s.executeId)
//			return
//		default:
//		}
//		task, ok := <-s.tasks
//		if !ok {
//			// 通道关闭，退出
//			break
//		}
//		// 开启新goroutine 执行
//		go func(t scheduledTask) {
//			if exec, exist := s.executors[string(t.executorType)]; !exist {
//				log.Printf("cron-center: scheduler[%d] 任务类型%s不存在执行器",
//					s.executeId, t.executorType)
//				return
//			} else {
//				// 执行任务
//				s.Exec(ctx, exec, *t.task)
//			}
//		}(task)
//	}
//}

//func (s *Scheduler) preempted() {
//	ctx, cancel := context.WithTimeout(context.Background(), s.dbTimeout)
//	// 抢占任务
//	tasks, err := s.storage.Preempt(ctx)
//	cancel()
//	if err != nil {
//		log.Printf("cron-center: scheduler[%d] 抢占任务失败：%s", s.executeId, err)
//		return
//	}
//	for _, task := range tasks {
//		s.tasks <- scheduledTask{
//			task:         &task,
//			executeId:    s.executeId,
//			executorType: task.Type,
//		}
//	}
//	cancel()
//}
//
//// Exec 刷新任务, 自动续约
//func (s *Scheduler) Exec(ctx context.Context, exe executor.Executor, t task.Task) {
//	exeCtx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	log.Printf("cron-center: scheduler[%d] 开始执行任务%s", s.executeId, t.Name)
//	go func(ctx context.Context) {
//		// 自动续约
//		ticker := time.NewTicker(s.refreshInterval)
//		defer ticker.Stop()
//		for {
//			select {
//			// 任务执行结束，取消续约
//			case <-ctx.Done():
//				log.Printf("cron-center: scheduler[%d] 任务%s自动续约结束", s.executeId, t.Name)
//				return
//			case <-ticker.C:
//				// 续约 不用关系是否续约成功，如果失败，其他实例会继续抢占执行
//				_ = s.storage.UpdateUtime(ctx, t.TaskId)
//			}
//		}
//	}(ctx)
//	err := exe.Exec(exeCtx, t)
//	if err != nil {
//		log.Printf("cron-center: scheduler[%d] 执行任务失败：%s", s.executeId, err)
//	}
//	// 任务执行结束，更新状态 更新下一次执行时间
//	nextTime := t.Next(time.Now())
//	dbCtx, cancel := context.WithTimeout(context.Background(), s.dbTimeout)
//	defer cancel()
//	err = s.storage.UpdateNextTime(dbCtx, t.TaskId, nextTime)
//	if err != nil {
//		log.Printf("cron-center: scheduler[%d] 更新任务失败执行时间失败：%s", s.executeId, err)
//	}
//}
