package executor

import (
	"context"
	"github.com/alehua/cron-center/internal/task"
)

type ExecType int

const (
	GoExecutorType ExecType = iota
	PythonExecutorType
	ShellExecutorType
	HttpExecutorType
)

// Executor 执行器抽象
type Executor interface {
	Name() string
	Exec(ctx context.Context, j task.Task) error
}
