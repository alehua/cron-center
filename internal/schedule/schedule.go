package schedule

import (
	"context"
	"github.com/alehua/cron-center/internal"
	"github.com/alehua/cron-center/internal/executor"
	"github.com/alehua/cron-center/internal/storage"
	"github.com/bwmarrin/snowflake"
	"log"
	"time"
)

// Scheduler 调度器
type Scheduler struct {
	executeId       int64
	tasks           chan scheduledTask
	executors       map[string]executor.Executor
	preemptInterval time.Duration   // 抢占任务间隔
	refreshInterval time.Duration   // 续约间隔
	storage         storage.Storage // 存储接口

	dbTimeout time.Duration // 数据库查询超时时间
	stop      chan struct{} // 停止信号
	stopFunc  func()
}

func NewScheduler(
	preemptInterval time.Duration,
	refreshInterval time.Duration,
	storage storage.Storage,
	dbTimeout time.Duration,
	maxTaskLen int8) *Scheduler {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
	executeId := node.Generate().Int64()
	s := &Scheduler{
		executeId:       executeId,
		tasks:           make(chan scheduledTask, maxTaskLen),
		executors:       make(map[string]executor.Executor),
		preemptInterval: preemptInterval,
		refreshInterval: refreshInterval,
		storage:         storage,
		dbTimeout:       dbTimeout,
		stop:            make(chan struct{}),
	}
	s.stopFunc = func() {
		close(s.tasks)
	}
	return s
}

type scheduledTask struct {
	task         *internal.Task
	executeId    int64
	executorType internal.TaskType
	stopped      bool
}

// Start 开始调度。当被取消，或者超时的时候，就会结束调度
func (s *Scheduler) Start(ctx context.Context) error {
	// 启动任务执行
	go s.execute(ctx)
	// 抢占任务间隔
	tickerP := time.NewTicker(s.preemptInterval)
	defer tickerP.Stop()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		select {
		case <-tickerP.C:
			log.Printf("cron-center: scheduler[%d] 开始进行任务抢占", s.executeId)
			go s.preempted()
		case <-s.stop:
			log.Printf("cron-center: scheduler[%d] 停止任务抢占", s.executeId)
			return nil
		default:
		}
	}
}

func (s *Scheduler) execute(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("cron-center: scheduler[%d] 停止任务执行", s.executeId)
			return
		default:
		}
		task, ok := <-s.tasks
		if !ok {
			// 通道关闭，退出
			break
		}
		// 开启新goroutine 执行
		go func(t scheduledTask) {
			if exec, exist := s.executors[string(t.executorType)]; !exist {
				log.Printf("cron-center: scheduler[%d] 任务类型%s不存在执行器",
					s.executeId, t.executorType)
				return
			} else {
				// 执行任务
				s.Exec(ctx, exec, *t.task)
			}
		}(task)
	}
}

func (s *Scheduler) preempted() {
	ctx, cancel := context.WithTimeout(context.Background(), s.dbTimeout)
	// 抢占任务
	tasks, err := s.storage.Preempt(ctx)
	cancel()
	if err != nil {
		log.Printf("cron-center: scheduler[%d] 抢占任务失败：%s", s.executeId, err)
		return
	}
	for _, task := range tasks {
		s.tasks <- scheduledTask{
			task:         &task,
			executeId:    s.executeId,
			executorType: task.Type,
		}
	}
	cancel()
}

// Exec 刷新任务, 自动续约
func (s *Scheduler) Exec(ctx context.Context, exe executor.Executor, t internal.Task) {
	exeCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log.Printf("cron-center: scheduler[%d] 开始执行任务%s", s.executeId, t.Name)
	go func(ctx context.Context) {
		// 自动续约
		ticker := time.NewTicker(s.refreshInterval)
		defer ticker.Stop()
		for {
			select {
			// 任务执行结束，取消续约
			case <-ctx.Done():
				log.Printf("cron-center: scheduler[%d] 任务%s自动续约结束", s.executeId, t.Name)
				return
			case <-ticker.C:
				// 续约 不用关系是否续约成功，如果失败，其他实例会继续抢占执行
				_ = s.storage.UpdateUtime(ctx, t.TaskId)
			}
		}
	}(ctx)
	err := exe.Exec(exeCtx, t)
	if err != nil {
		log.Printf("cron-center: scheduler[%d] 执行任务失败：%s", s.executeId, err)
	}
	// 任务执行结束，更新状态 更新下一次执行时间
	nextTime := t.Next(time.Now())
	dbCtx, cancel := context.WithTimeout(context.Background(), s.dbTimeout)
	defer cancel()
	err = s.storage.UpdateNextTime(dbCtx, t.TaskId, nextTime)
	if err != nil {
		log.Printf("cron-center: scheduler[%d] 更新任务失败执行时间失败：%s", s.executeId, err)
	}
}
