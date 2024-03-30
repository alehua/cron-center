package schedule

import (
	"context"
	"github.com/alehua/cron-center/internal/domain"
	"github.com/alehua/cron-center/internal/storage"
	"time"
)

// CronTaskService 定时任务服务接口
type CronTaskService interface {
	Preempt(ctx context.Context) ([]domain.Task, error)       // 抢占一个任务
	ResetNextTime(ctx context.Context, job domain.Task) error // 重置任务下次执行时间
	AddJob(ctx context.Context, j domain.Task) error          // 新增任务
}

type cronJobService struct {
	storage         storage.Storage
	refreshInterval time.Duration
	currentJob      domain.Task
}

func (s *cronJobService) AddJob(ctx context.Context, j domain.Task) error {
	next := j.Next(time.Now())
	j.NextTime = next.UnixMilli()
	return s.storage.Insert(ctx, j)
}

func (s *cronJobService) Preempt(ctx context.Context) ([]domain.Task, error) {
	j, err := s.storage.Preempt(ctx)
	if err != nil {
		return []domain.Task{}, err
	}
	s.currentJob = j
	ch := make(chan struct{})
	go func() {
		ticker := time.NewTicker(s.refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ch:
				// 退出续约循环
				return
			case <-ticker.C:
				s.refresh(j.TaskId)
			}
		}
	}()
	// 只能调用一次，也就是放弃续约。这时候要把状态还原回去
	//s.currentJob.CancelFunc = func() {
	//	close(ch)
	//	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//	defer cancel()
	//	err := s.repo.Release(ctx, s.currentJob.Id, s.currentJob.Utime)
	//	if err != nil {
	//		s.l.Error("释放任务失败",
	//			logger.Error(err),
	//			logger.Int64("id", s.currentJob.Id))
	//	}
	//}
	return s.currentJob, nil
}

func (s *cronJobService) refresh(id int64) {
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
	//// utime, err := s.storage.UpdateUtime(ctx, id)
	//if err != nil {
	//
	//}
	// s.currentJob. = utime
}

func (s *cronJobService) ResetNextTime(ctx context.Context, task domain.Task) error {
	// 计算下一次的时间
	t := task.Next(time.Now())
	// 我们认为这是不需要继续执行了
	if !t.IsZero() {
		return s.storage.UpdateNextTime(ctx, task.TaskId, t)
	}
	return nil
}
