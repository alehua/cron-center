package main

import (
	"context"
	"github.com/alehua/cron-center/internal/pkg/logger"
	"github.com/alehua/cron-center/internal/schedule"
	"github.com/alehua/cron-center/internal/storage"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"net"
)

func main() {
	l := InitLogger()
	db := InitDB(l)
	instantId := GetOutboundIP() // 计算本级IP作为实例的ID
	st := storage.NewTaskStorage(db, instantId,
		storage.WithRefreshLimit(10))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go st.Preempt(ctx)
	go st.AutoRefresh(ctx)

	sche := schedule.NewScheduler(st)
	if err := sche.Start(ctx); err != nil {
		panic(err)
	}
}

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

// 函数实现类型 实现gorm Writer
type goormLoggerFunc func(msg string, fields ...logger.Field)

func (g goormLoggerFunc) Printf(s string, i ...interface{}) {
	g(s, logger.Field{Key: "args", Val: i})
}

// GetOutboundIP 获得对外发送消息的 IP 地址
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func InitLogger() logger.Logger {
	return logger.NewZapLogger(zap.L())
}
