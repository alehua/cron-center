package main

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	//db := InitDB()
	//dao := dao.NewGORMTaskDAO(db, 1)
	//st := storage.NewTaskStorage(dao)
	//sche := schedule.NewScheduler(1*time.Second, 1*time.Second, st, 1*time.Second, 100)
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	//if err := sche.Start(ctx); err != nil {
	//	panic(err)
	//}
}

func InitDB() *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:root@tcp(localhost:13316)/test"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}
