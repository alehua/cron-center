package schedule

import (
	"context"
	"github.com/alehua/cron-center/internal"
	"github.com/alehua/cron-center/internal/executor"
	"github.com/alehua/cron-center/internal/storage"
	"golang.org/x/sync/semaphore"
	"log"
	"time"
)

// Scheduler 调度器
type Scheduler struct {
	executeId       int64
	tasks           map[string]scheduledTask
	executors       map[string]executor.Executor
	limiter         *semaphore.Weighted
	preemptInterval time.Duration   // 抢占任务间隔
	refreshInterval time.Duration   // 续约间隔
	storage         storage.Storage // 存储接口

	dbTimeout time.Duration   // 数据库查询超时时间
	stop      <-chan struct{} // 停止信号
}

type scheduledTask struct {
	task      *internal.Task
	executeId int64
	executor  executor.Executor
	stopped   bool
}

//func (s *Scheduler) RegisterJob(ctx context.Context, j CronJob) error {
//	return s.svc.AddJob(ctx, j)
//}

func (s *Scheduler) RegisterExecutor(exec executor.Executor) {
	s.executors[exec.Name()] = exec
}

// Start 开始调度。当被取消，或者超时的时候，就会结束调度
func (s *Scheduler) Start(ctx context.Context) error {
	// 抢占任务间隔
	tickerP := time.NewTicker(s.preemptInterval)
	defer tickerP.Stop()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := s.limiter.Acquire(ctx, 1)
		if err != nil {
			return err
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
		if _, ok := s.tasks[task.Id]; !ok {
			s.addTask(ctx, task)
		} else {
			log.Printf("cron-center: scheduler[%d] 任务%s已经存在，跳过", s.executeId, task.Id)
		}
	}
	cancel()

}

// Refresh 刷新任务, 自动续约
func (s *Scheduler) Refresh(ctx context.Context) {
	timer := time.NewTicker(s.refreshInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			//tasks, _ := eorm.NewSelector[TaskInfo](s.db).
			//	From(eorm.TableOf(&TaskInfo{}, "t1")).
			//	Where(eorm.C("SchedulerStatus").EQ(storage.EventTypePreempted).
			//		And(eorm.C("OccupierId").EQ(s.storageId))).
			//	GetMulti(ctx)
			//for _, t := range tasks {
			//	go s.refresh(ctx, t.Id, t.Epoch, t.CandidateId)
			//}
		case <-s.stop:
			//log.Printf("ecron: storage[%d]关闭，停止所有task的自动续约", s.storageId)
			//return
		}
	}
}
