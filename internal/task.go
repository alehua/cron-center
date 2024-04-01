package internal

import (
	"github.com/robfig/cron/v3"
	"time"
)

type TaskType string

const (
	TypeHTTP   = "http_task"
	TypePython = "python_task"
	TypeShell  = "shell_task"
	TypeLocal  = "local_task"
)

type Config struct {
	Name       string
	Cron       string
	Type       TaskType
	Cmd        string
	Parameters string
	MaxTime    time.Duration // 任务的最大执行时间
}

// Task 任务的执行信息
type Task struct {
	Config             // 任务的配置信息
	TaskId   int64     // 任务的唯一ID
	NextTime time.Time // 下次执行时间
	Status   int       // 任务的状态
	Version  int64     // 任务的执行序号
}

// Next 获取下次执行时间
func (task *Task) Next(t time.Time) time.Time {
	expr := cron.NewParser(cron.Second | cron.Minute |
		cron.Hour | cron.Dom |
		cron.Month | cron.Dow |
		cron.Descriptor)
	s, _ := expr.Parse(task.Cron)
	return s.Next(t)
}
