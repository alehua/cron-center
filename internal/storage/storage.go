package storage

import (
	"context"
	"errors"
	"github.com/alehua/cron-center/internal/task"
	"gorm.io/gorm"
	"log"
	"time"
)

const (
	// TaskStatusWaiting 等待被调度
	TaskStatusWaiting = iota
	// TaskStatusRunning 已经被 goroutine 抢占了
	TaskStatusRunning
	// TaskStatusEnd 不再需要调度了，执行完毕。
	TaskStatusEnd
)

type TaskInfo struct {
	Id              int64 `gorm:"auto_increment,primary_key"`
	Name            string
	SchedulerStatus string
	Version         int64
	Cron            string
	Type            string
	InstanceId      int32
	MaxExecTime     int32
	Config          string
	CreateTime      int64
	UpdateTime      int64
}

type TaskExecution struct {
	Id            int64 `gorm:"auto_increment,primary_key"`
	TaskId        int64
	ExecuteStatus string
	CreateTime    int64
	UpdateTime    int64
}

// ******** 初始化方法 **********

type TaskInfoStorage struct {
	db              *gorm.DB
	refreshInterval time.Duration // 续约间隔
	preemptInterval time.Duration // 抢占间隔
	instanceId      int32
	limit           int
	events          chan Event
	stop            chan struct{}
}

type Option func(t *TaskInfoStorage)

func NewTaskStorage(db *gorm.DB, id int32, opts ...Option) Storager {
	dao := &TaskInfoStorage{
		db:              db,
		instanceId:      id,
		limit:           10,
		refreshInterval: 5 * time.Second,
		preemptInterval: 10 * time.Second,
		events:          make(chan Event),
		stop:            make(chan struct{}),
	}
	for _, opt := range opts {
		opt(dao)
	}
	return dao
}

func WithPreemptInterval(t time.Duration) Option {
	return func(dao *TaskInfoStorage) {
		dao.preemptInterval = t
	}
}

func WithRefreshLimit(limit int) Option {
	return func(dao *TaskInfoStorage) {
		dao.limit = limit
	}
}

func WithRefreshInterval(t time.Duration) Option {
	return func(dao *TaskInfoStorage) {
		dao.refreshInterval = t
	}
}

// ************** 接口的实现 ***********

func (dao *TaskInfoStorage) Events(ctx context.Context, taskEvents <-chan task.Event) <-chan Event {
	go func() {
		for {
			select {
			case <-ctx.Done():
			case event := <-taskEvents:
				switch event.Type {
				case task.EventTypeRunning:
					log.Println("storage 收到 task执行中信号")
					_ = dao.UpdateTaskStatus(ctx, event.TaskId, EventTypePreempted, EventTypeRunning)
				case task.EventTypeSuccess:
					_ = dao.UpdateTaskStatus(ctx, event.TaskId, EventTypePreempted, EventTypeSuccess)
				case task.EventTypeFailed:
					_ = dao.UpdateTaskStatus(ctx, event.TaskId, EventTypePreempted, EventTypeFail)
				}
			}
		}
	}()
	return dao.events
}

// UpdateTaskStatus 更新任务状态
func (dao *TaskInfoStorage) UpdateTaskStatus(ctx context.Context, taskId int64, old, new string) error {
	res := dao.db.WithContext(ctx).Model(&TaskExecution{}).
		Where("id = ? AND execute_status = ?", taskId, old).
		Updates(map[string]any{
			"update_time":    time.Now().UnixMilli(),
			"execute_status": new,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("任务状态不对")
	}
	return nil
}

func (dao *TaskInfoStorage) Get(ctx context.Context, id int64) (*task.Task, error) {
	var info TaskInfo
	err := dao.db.WithContext(ctx).Model(&TaskInfo{}).
		Where("id = ?", id).First(&info).Error
	if err != nil {
		return &task.Task{}, err
	}
	return dao.toTask(info), nil
}

func (dao *TaskInfoStorage) Insert(ctx context.Context, t *task.Task) error {
	info := dao.toTaskInfo(t)
	now := time.Now().UnixMilli()
	info.CreateTime = now
	info.UpdateTime = now
	info.SchedulerStatus = EventTypeCreated
	return dao.db.WithContext(ctx).Create(&info).Error
}

func (dao *TaskInfoStorage) Preempt(ctx context.Context) {
	// 抢占任务间隔
	tickerP := time.NewTicker(dao.preemptInterval)
	defer tickerP.Stop()
	for {
		select {
		case <-tickerP.C:
			go func() {
				tasks, err := dao.preempted(ctx)
				if err != nil {
					log.Printf("preempted error: %v", err) // 这里要报警
				}
				for _, item := range tasks {
					// 通知调度器
					preemptedEvent := Event{
						Type: EventTypePreempted,
						Task: &task.Task{
							Config: task.Config{Name: item.Name, Cron: item.Cron, Type: item.Type},
							TaskId: item.Id,
						},
					}
					dao.events <- preemptedEvent
				}
			}()
		case <-dao.stop:
			return
		default:
		}
	}
}

// Preempt 执行一次抢占
func (dao *TaskInfoStorage) preempted(ctx context.Context) ([]TaskInfo, error) {
	db := dao.db.WithContext(ctx)
	var TaskInfos []TaskInfo
	// 每一个循环都重新计算 time.Now
	now := time.Now()
	var tmp []TaskInfo
	// GORM 多条件查询
	// 条件1: 下一次执行时间小于当前时间，并且状态是等待中
	cond1 := db.Where("next_time <= ? AND status = ?", now, TaskStatusWaiting)
	// 条件2: 状态是运行态 (某一次续约失败，utime没有变)
	// 10分钟内没有抢到, 认为任务已经过期
	const threshold = 10 * time.Minute
	ddl := now.Add(-1 * threshold).UnixMilli()
	cond2 := db.Where("utime <= ? AND status = ?", ddl, TaskStatusRunning)

	err := db.Model(&TaskInfo{}).
		Where(cond1.Or(cond2)).
		Limit(dao.limit).Find(&tmp).Error
	if err != nil {
		// 数据库有问题
		return TaskInfos, err
	}
	// 开始抢占, 通过version来保证原子性 upsert语义
	for _, item := range tmp {
		res := db.Model(&TaskInfo{}).
			Where("id = ? AND version = ?", item.Id, item.Version).
			Updates(map[string]any{
				"utime":   now.UnixMilli(),
				"version": item.Version + 1,
				"status":  TaskStatusRunning,
			})
		if res.Error != nil {
			continue // 数据库错误, 记录日志, 继续下一个
		}
		// 抢占成功
		if res.RowsAffected == 1 {
			TaskInfos = append(TaskInfos, item)
		}
	}
	return TaskInfos, nil
}

// AutoRefresh 自动续约
func (dao *TaskInfoStorage) AutoRefresh(ctx context.Context) {
	// 续约任务间隔
	timer := time.NewTicker(dao.refreshInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			var tasks []TaskInfo
			err := dao.db.WithContext(ctx).Model(&TaskInfo{}).
				Where("SchedulerStatus = ? AND instance_id = ?",
					EventTypePreempted, dao.instanceId).
				Find(&tasks).Error
			if err != nil {
				log.Printf("cron: storage[%d]自动续约失败，%v", dao.instanceId, err)
				continue
			}
			for _, t := range tasks {
				go dao.refresh(ctx, t.Id)
			}
		case <-dao.stop:
			log.Printf("cron: storage[%d]关闭，停止所有task的自动续约", dao.instanceId)
			return
		}
	}
}

func (dao *TaskInfoStorage) refresh(ctx context.Context, id int64) {
	now := time.Now().UnixMilli()
	err := dao.db.WithContext(ctx).Model(&TaskInfo{}).
		Where("id=?", id).Updates(map[string]any{
		"utime": now,
	}).Error
	if err != nil {
		log.Printf("cron: storage[%d]自动续约失败，%v", dao.instanceId, err)
	}
	// 根据error判断是否需要重新重试
}

func (dao *TaskInfoStorage) Release(ctx context.Context, id int64) error {
	// 释放是的时候判断是否自己抢占的, 确保更新时间和自己强制时候一致
	res := dao.db.WithContext(ctx).Model(&TaskInfo{}).
		Where("id = ? AND instance_id = ?", id, dao.instanceId).Updates(map[string]any{
		"status": TaskStatusEnd,
		"utime":  time.Now().UnixMilli(),
	})
	if res.RowsAffected == 0 {
		// 任务已经不是自己的, 无须释放。 理论上是不会出现这种情况
		return nil
	}
	return res.Error
}

// AddExecution 创建一条执行记录
func (dao *TaskInfoStorage) AddExecution(ctx context.Context, taskId int64) error {
	var t = TaskExecution{
		TaskId:        taskId,
		ExecuteStatus: task.EventTypeInit,
		CreateTime:    time.Now().Unix(),
		UpdateTime:    time.Now().Unix(),
	}
	return dao.db.WithContext(ctx).Model(&TaskExecution{}).Create(&t).Error
}

func (dao *TaskInfoStorage) toTask(info TaskInfo) *task.Task {
	return &task.Task{
		Config: task.Config{
			Name:    info.Name,
			Cron:    info.Cron,
			Type:    info.Type,
			MaxTime: time.Duration(info.MaxExecTime),
		},
		TaskId:  info.Id,
		Version: info.Version,
	}
}

func (dao *TaskInfoStorage) toTaskInfo(t *task.Task) *TaskInfo {
	return &TaskInfo{
		Name:        t.Name,
		Version:     t.Version,
		Cron:        t.Cron,
		Type:        t.Cron,
		MaxExecTime: int32(t.MaxTime.Seconds()),
	}
}
