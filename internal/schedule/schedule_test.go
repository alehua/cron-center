package schedule

import (
	"context"
	"github.com/alehua/cron-center/internal/executor/local"
	"github.com/alehua/cron-center/internal/storage"
	storagemocks "github.com/alehua/cron-center/internal/storage/mocks"
	"github.com/alehua/cron-center/internal/task"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func TestExecuteLoop(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	st := func(ctrl *gomock.Controller) storage.Storager {
		res := storagemocks.NewMockStorager(ctrl)
		return res
	}(ctl)

	execTask := &scheduledTask{
		task:       &task.Task{},
		executor:   local.NewLocalFuncExecutor(),
		taskEvents: make(chan task.Event),
	}
	sche := NewScheduler(st)
	_ = sche.readyTasks.Enqueue(context.TODO(), execution{
		scheduledTask: execTask,
		time:          time.Now().Add(5 * time.Second),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := sche.executeLoop(ctx)
	assert.NoError(t, err)
}
