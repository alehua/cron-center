package storage

import (
	"context"
	"github.com/alehua/cron-center/internal"
	"github.com/alehua/cron-center/internal/storage/dao"
	"github.com/alehua/ekit/slice"
	"time"
)

type Storage interface {
	// Preempt 抢占一个任务
	Preempt(ctx context.Context) ([]internal.Task, error)
	UpdateNextTime(ctx context.Context, id int64, t time.Time) error
	UpdateUtime(ctx context.Context, id int64) (int64, error)
	// Release 释放一个任务
	Release(ctx context.Context, id, utime int64) error
	// Insert 插入一个任务
	Insert(ctx context.Context, t internal.Task) error
}

type TaskStorage struct {
	dao dao.GORMTaskDAO
}

func (ts *TaskStorage) Preempt(ctx context.Context) ([]internal.Task, error) {
	data, err := ts.dao.Preempt(ctx)
	if err != nil {
		return []internal.Task{}, err
	}
	return slice.Map[dao.Task, internal.Task](data, func(idx int, src dao.Task) internal.Task {
		return ts.ToInternal(src)
	}), err
}

func (ts *TaskStorage) UpdateNextTime(ctx context.Context, id int64, t time.Time) error {
	return ts.dao.UpdateNextTime(ctx, id, t)
}

func (ts *TaskStorage) UpdateUtime(ctx context.Context, id int64) (int64, error) {
	return ts.dao.UpdateUtime(ctx, id)
}

func (ts *TaskStorage) Release(ctx context.Context, id, utime int64) error {
	return ts.dao.Release(ctx, id, utime)
}

func (ts *TaskStorage) Insert(ctx context.Context, t internal.Task) error {
	return ts.dao.Insert(ctx, ts.ToEntity(t))
}

func (ts *TaskStorage) ToEntity(t internal.Task) dao.Task {
	return dao.Task{
		Name:       t.Name,
		Cron:       t.Cron,
		Cmd:        t.Cmd,
		Parameters: t.Parameters,
		Id:         t.TaskId,
		NextTime:   t.NextTime,
		Status:     t.Status,
		Version:    t.Version,
	}
}

func (ts *TaskStorage) ToInternal(t dao.Task) internal.Task {
	config := internal.Config{
		Name:       t.Name,
		Cron:       t.Cron,
		Cmd:        t.Cmd,
		Parameters: t.Parameters,
	}
	return internal.Task{
		Config:   config,
		TaskId:   t.Id,
		NextTime: t.NextTime,
		Status:   t.Status,
		Version:  t.Version,
	}

}

//Events(ctx context.Context, taskEvents <-chan internal.Event) (<-chan Event, error)
//JobDAO
