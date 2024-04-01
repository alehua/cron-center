package executor

import (
	"context"
	"github.com/alehua/cron-center/internal"
)

// Executor 执行器抽象
type Executor interface {
	Name() string
	Exec(ctx context.Context, j internal.Task) error
}
