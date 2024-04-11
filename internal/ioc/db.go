package ioc

import (
	"github.com/alehua/cron-center/internal/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func InitDB(l logger.Logger) *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:root@tcp(localhost:13317)/cron"),
		&gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true, // 使用单数表名
			},
		})
	if err != nil {
		panic(err)
	}
	return db
}
