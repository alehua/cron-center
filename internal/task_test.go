package internal

import (
	"testing"
	"time"
)

func TestTaskNextTime(t *testing.T) {
	testCases := []struct {
		task     Task
		taskName string
	}{
		{
			task: Task{
				Config: Config{
					Cron: "0 0/2 * * * ?",
				},
				TaskId: 1,
			},
			taskName: "分钟任务",
		},
		{
			task: Task{
				Config: Config{
					Cron: "*/2 * * * * ?",
				},
				TaskId: 2,
			},
			taskName: "秒任务",
		},
		{
			task: Task{
				Config: Config{
					Cron: "0 0 12 * * ?",
				},
				TaskId: 3,
			},
			taskName: "天级别任务",
		},
	}
	for _, task := range testCases {
		t.Run(task.taskName, func(t *testing.T) {
			t.Log("当前时间:", time.Now().Format(time.DateTime))
			nextTime := task.task.Next(time.Now())
			t.Log("next时间:", nextTime.Format(time.DateTime))
		})
	}
}
