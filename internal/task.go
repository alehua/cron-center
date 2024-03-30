package internal

import "time"

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
	Parameters string
	MaxTime    time.Duration // 任务的最大执行时间
}

// Task 任务的执行信息
type Task struct {
	Config       // 任务的配置信息
	TaskId int64 // 任务的唯一ID
}
