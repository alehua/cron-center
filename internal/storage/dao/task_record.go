package dao

import (
	"context"
	"github.com/alehua/cron-center/internal/task"
	"gorm.io/gorm"
	"time"
)

type TaskExecution struct {
	Id            int64 `gorm:"auto_increment,primary_key"`
	TaskId        int64
	ExecuteStatus string
	CreateTime    int64
	UpdateTime    int64
}

type GORMTaskRecordDAO struct {
	db *gorm.DB
}

// AddExecution 创建一条执行记录
func (g *GORMTaskRecordDAO) AddExecution(ctx context.Context, taskId int64) error {
	var t = TaskExecution{
		TaskId:        taskId,
		ExecuteStatus: task.EventTypeInit,
		CreateTime:    time.Now().Unix(),
		UpdateTime:    time.Now().Unix(),
	}
	return g.db.WithContext(ctx).Create(&t).Error
}
