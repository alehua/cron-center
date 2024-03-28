package task

import (
	"github.com/robfig/cron/v3"
	"time"
)

type Type string

type EventType string

const (
	// EventTypePreempted 当前调度节点已经抢占了这个任务
	EventTypePreempted = "preempted"
	// EventTypeRunnable 已经到了可运行的时间点
	EventTypeRunnable = "runnable"
	// EventTypeRunning 已经找到了目标节点，并且正在运行
	EventTypeRunning = "running"
	// EventTypeFailed 任务运行失败
	EventTypeFailed = "failed"
	// EventTypeSuccess 任务运行成功
	EventTypeSuccess = "success"
	// EventTypeKnown 任务默认状态
	EventTypeKnown = "known"
)

type Config struct {
	Name       string
	Cron       string
	Cmd        string
	Parameters string
}

// Task 任务的信息
type Task struct {
	Config
	TaskId   int64
	NextTime int64
	Status   int
	Version  int64
}

func (task *Task) Next(t time.Time) time.Time {
	expr := cron.NewParser(cron.Second | cron.Minute |
		cron.Hour | cron.Dom |
		cron.Month | cron.Dow |
		cron.Descriptor)
	s, _ := expr.Parse(task.Cron)
	return s.Next(t)
}
