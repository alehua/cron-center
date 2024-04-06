package executor

import (
	"context"
	"github.com/alehua/cron-center/internal/task"
)

type ExecType string

const (
	GoExecutorType     ExecType = "local"
	PythonExecutorType          = "python"
	ShellExecutorType           = "shell"
	HttpExecutorType            = "http"
)

// Executor 执行器抽象
type Executor interface {
	Name() string
	Exec(ctx context.Context, j task.Task) error
}
