package local

import (
	"context"
	"errors"

	"github.com/alehua/cron-center/internal/domain"
)

type FuncExecutor struct {
	funcs map[string]func(ctx context.Context, j domain.Task) error
}

func NewLocalFuncExecutor() *FuncExecutor {
	return &FuncExecutor{funcs: make(map[string]func(ctx context.Context, j domain.Task) error)}
}

func (l *FuncExecutor) AddLocalFunc(name string,
	fn func(ctx context.Context, j domain.Task) error) {
	l.funcs[name] = fn
}

func (l *FuncExecutor) Name() string {
	return "local"
}

func (l *FuncExecutor) Exec(ctx context.Context, j domain.Task) error {
	fn, ok := l.funcs[j.Name]
	if !ok {
		return errors.New("是不是忘记注册本地方法了？")
	}
	return fn(ctx, j)
}
