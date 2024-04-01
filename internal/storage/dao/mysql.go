package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

const (
	// TaskStatusWaiting 等待被调度
	TaskStatusWaiting = iota
	// TaskStatusRunning 已经被 goroutine 抢占了
	TaskStatusRunning
	// TaskStatusEnd 不再需要调度了，比如说被终止了，或者被删除了。
	TaskStatusEnd
	// TaskStatusDone 任务已经执行完毕
	TaskStatusDone
)

type TaskDAO interface {
	// Preempt 抢占任务
	Preempt(ctx context.Context) ([]Task, error)
	UpdateNextTime(ctx context.Context, id int64, t time.Time) error
	UpdateUtime(ctx context.Context, id int64) error
	// Release 释放一个任务
	Release(ctx context.Context, id, utime int64) error
	// Insert 插入一个任务
	Insert(ctx context.Context, j Task) error
}

type GORMTaskDAO struct {
	db        *gorm.DB
	storageId int64
}

func (dao *GORMTaskDAO) Insert(ctx context.Context, j Task) error {
	now := time.Now().UnixMilli()
	j.Ctime = now
	j.Utime = now
	return dao.db.WithContext(ctx).Create(&j).Error
}

func NewGORMTaskDAO(db *gorm.DB, storageId int64) TaskDAO {
	return &GORMTaskDAO{db: db, storageId: storageId}
}

func (dao *GORMTaskDAO) Release(ctx context.Context, id, utime int64) error {
	// 释放是的时候判断是否自己抢占的, 确保更新时间和自己强制时候一致
	res := dao.db.WithContext(ctx).Model(&Task{}).
		Where("id = ? AND utime = ?", id, utime).Updates(map[string]any{
		"status": TaskStatusWaiting,
		"utime":  time.Now().UnixMilli(),
	})
	if res.RowsAffected == 0 {
		// 任务已经不是自己的, 无须释放。 理论上是不会出现这种情况
		return nil
	}
	return res.Error
}

func (dao *GORMTaskDAO) UpdateUtime(ctx context.Context, id int64) error {
	now := time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Model(&Task{}).
		Where("id=?", id).Updates(map[string]any{
		"utime": now,
	}).Error
}

// Preempt 执行一次抢占
func (dao *GORMTaskDAO) Preempt(ctx context.Context) ([]Task, error) {
	db := dao.db.WithContext(ctx)
	var tasks []Task
	// 每一个循环都重新计算 time.Now
	now := time.Now()
	var tmp []Task
	// GORM 多条件查询
	// 条件1: 下一次执行时间小于当前时间，并且状态是等待中
	cond1 := db.Where("next_time <= ? AND status = ?", now, TaskStatusWaiting)
	// 条件2: 状态是运行态 (某一次续约失败，utime没有变)
	// 10分钟内没有抢到, 认为任务已经过期
	const threshold = 10 * time.Minute
	ddl := now.Add(-1 * threshold).UnixMilli()
	cond2 := db.Where("utime <= ? AND status = ?", ddl, TaskStatusRunning)
	// 条件3: 当前任务拥有者主动放弃, 状态是TaskStatusDone
	cond3 := db.Where("status = ? AND executor = ?", TaskStatusDone, dao.storageId)

	err := db.Model(&Task{}).Where(cond1.Or(cond2).Or(cond3)).Find(&tmp).Error
	if err != nil {
		// 数据库有问题
		return tasks, err
	}
	// 开始抢占, 通过version来保证原子性 upsert语义
	for _, item := range tmp {
		res := db.Model(&Task{}).
			Where("id = ? AND version = ?", item.Id, item.Version).
			Updates(map[string]any{
				"utime":   now.UnixMilli(),
				"version": item.Version + 1,
				"status":  TaskStatusRunning,
			})
		if res.Error != nil {
			continue
			// 数据库错误, 记录日志, 继续下一个
		}
		// 抢占成功
		if res.RowsAffected == 1 {
			tasks = append(tasks, item)
		}
	}
	return tasks, nil
}

func (dao *GORMTaskDAO) UpdateNextTime(ctx context.Context, id int64, t time.Time) error {
	return dao.db.WithContext(ctx).Model(&Task{}).
		Where("id=?", id).Updates(map[string]any{
		"utime":     time.Now().UnixMilli(),
		"next_time": t.UnixMilli(),
	}).Error
}

type Task struct {
	Id         int64 `gorm:"primaryKey,autoIncrement"`
	Name       string
	Cron       string
	Cmd        string
	Parameters string
	Version    int64
	NextTime   int64 `gorm:"index"`
	Status     int
	Ctime      int64
	Utime      int64 `gorm:"index"`
}
