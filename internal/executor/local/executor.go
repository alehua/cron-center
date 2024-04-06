package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/alehua/cron-center/internal/task"
)

type FuncExecutor struct {
	funcs map[string]func(ctx context.Context) error
}

func NewLocalFuncExecutor() *FuncExecutor {
	executor := &FuncExecutor{
		funcs: make(map[string]func(ctx context.Context) error),
	}
	// 在这里添加任务
	executor.AddLocalFunc("demo", Demo)
	return executor
}

func (l *FuncExecutor) AddLocalFunc(name string, fn func(ctx context.Context) error) {
	l.funcs[name] = fn
}

func (l *FuncExecutor) Name() string {
	return "local"
}

func (l *FuncExecutor) Exec(ctx context.Context, t task.Task) error {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				fmt.Printf("%s 执行出错, %s\n", t.Name, err.Error())
			} else {
				fmt.Printf("%s 执行出错", t.Name)
			}
		}
	}()
	fn, ok := l.funcs[t.Name]
	if !ok {
		return errors.New("是不是忘记注册本地方法了？")
	}
	return fn(ctx)
}
