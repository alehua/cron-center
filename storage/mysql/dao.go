package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"
)

const (
	// 等待被调度
	jobStatusWaiting = iota
	// 已经被 goroutine 抢占了
	jobStatusRunning
	// 不再需要调度了，比如说被终止了，或者被删除了。
	jobStatusEnd
)

var ErrNoMoreJob = gorm.ErrRecordNotFound

type JobDAO interface {
	// Preempt 抢占一个任务
	Preempt(ctx context.Context) (Job, error)
	UpdateNextTime(ctx context.Context, id int64, t time.Time) error
	UpdateUtime(ctx context.Context, id int64) (int64, error)
	// Release 释放一个任务
	Release(ctx context.Context, id, utime int64) error
	// Insert 插入一个任务
	Insert(ctx context.Context, j Job) error
}

type GORMJobDAO struct {
	db *gorm.DB
}

func (dao *GORMJobDAO) Insert(ctx context.Context, j Job) error {
	now := time.Now().UnixMilli()
	j.Ctime = now
	j.Utime = now
	return dao.db.WithContext(ctx).Create(&j).Error
}

func NewGORMJobDAO(db *gorm.DB) JobDAO {
	return &GORMJobDAO{db: db}
}

func (dao *GORMJobDAO) Release(ctx context.Context, id, utime int64) error {
	// 释放是的时候判断是否自己抢占的, 确保更新时间和自己强制时候一致
	res := dao.db.WithContext(ctx).Model(&Job{}).
		Where("id = ? AND utime = ?", id, utime).Updates(map[string]any{
		"status": jobStatusWaiting,
		"utime":  time.Now().UnixMilli(),
	})
	if res.RowsAffected == 0 {
		// 任务已经不是自己的, 无须释放。 理论上是不会出现这种情况
		return nil
	}
	return res.Error
}

func (dao *GORMJobDAO) UpdateUtime(ctx context.Context, id int64) (int64, error) {
	now := time.Now().UnixMilli()
	return now, dao.db.WithContext(ctx).Model(&Job{}).
		Where("id=?", id).Updates(map[string]any{
		"utime": now,
	}).Error
}

func (dao *GORMJobDAO) Preempt(ctx context.Context) (Job, error) {
	db := dao.db.WithContext(ctx)
	for {
		// 每一个循环都重新计算 time.Now
		now := time.Now()
		var j Job
		const threshold = 10 * time.Minute
		ddl := now.Add(-1 * threshold).UnixMilli()
		err := db.Where(
			// 条件1: 下一次执行时间小于当前时间，并且状态是等待中
			db.Where("next_time <= ? AND status = ?", now, jobStatusWaiting).Or(
				// 条件2: 状态是运行态 (第一次续约就失败, 某一次续约失败，utime没有变)
				"utime <= ? AND status = ?", ddl, jobStatusRunning,
			),
		).First(&j).Error
		if err != nil {
			// 数据库有问题
			return Job{}, err
		}
		// 开始抢占, 通过version来保证原子性 upsert语义
		res := db.Model(&Job{}).
			Where("id = ? AND version=?", j.Id, j.Version).
			Updates(map[string]any{
				"utime":   now.UnixMilli(),
				"version": j.Version + 1,
				"status":  jobStatusRunning,
			})
		if res.Error != nil {
			// 数据库错误
			return Job{}, err
		}
		// 抢占成功
		if res.RowsAffected == 1 {
			return j, nil
		}
		// 没有抢占到，也就是同一时刻被人抢走了，那么就下一个循环
		// 如果多少次没有抢到, 退出循环
	}
}

func (dao *GORMJobDAO) UpdateNextTime(ctx context.Context, id int64, t time.Time) error {
	return dao.db.WithContext(ctx).Model(&Job{}).
		Where("id=?", id).Updates(map[string]any{
		"utime":     time.Now().UnixMilli(),
		"next_time": t.UnixMilli(),
	}).Error
}

type Job struct {
	Id         int64 `gorm:"primaryKey,autoIncrement"`
	Name       string
	Executor   string
	Cfg        string
	Expression string
	Version    int64
	NextTime   int64 `gorm:"index"`
	Status     int
	Ctime      int64
	Utime      int64 `gorm:"index"`
}
