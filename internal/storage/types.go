package storage

import (
	"context"
	"github.com/alehua/cron-center/internal/task"
)

const (
	// EventTypePreempted 抢占了一个任务
	EventTypePreempted = "preempted"
	EventTypeRunning   = "running"
	EventTypeSuccess   = "success"
	EventTypeFail      = "fail"
)

//go:generate mockgen -source=types.go -package=storagemocks -destination=mocks/types.mock.go Storager
type Storager interface {
	Events(ctx context.Context, taskEvents <-chan task.Event) <-chan Event
	TaskDAO
}

type TaskDAO interface {
	AddExecution(ctx context.Context, taskId int64) error

	Get(ctx context.Context, id int64) (*task.Task, error)
	Insert(ctx context.Context, t *task.Task) error
	Preempt(ctx context.Context)
	AutoRefresh(ctx context.Context)
	Release(ctx context.Context, id int64) error
}

type Status struct {
	ExpectStatus string
	UseStatus    string
}

type Event struct {
	Type string
	Task *task.Task
}
