package main

import (
	"context"
	"github.com/alehua/cron-center/internal/schedule"
	"github.com/alehua/cron-center/internal/storage"
	"github.com/alehua/cron-center/internal/storage/dao"
	"gorm.io/gorm"
	"time"

	"gorm.io/driver/mysql"
)

func main() {
	db := InitDB()
	dao := dao.NewGORMTaskDAO(db, 1)
	st := storage.NewTaskStorage(dao)
	sche := schedule.NewScheduler(1*time.Second, 1*time.Second, st, 1*time.Second, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := sche.Start(ctx); err != nil {
		panic(err)
	}
}

func InitDB() *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:root@tcp(localhost:13316)/webook"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	err = dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return db
}
