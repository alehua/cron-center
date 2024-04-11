package ioc

import (
	"github.com/alehua/cron-center/internal/schedule"
	"github.com/alehua/cron-center/internal/storage"
	"net"
)

func Schedule() *schedule.Scheduler {
	l := InitLogger()
	db := InitDB(l)
	instantId := getOutboundIP() // 计算本级IP作为实例的ID
	st := storage.NewTaskStorage(db, instantId, storage.WithRefreshLimit(10))
	return schedule.NewScheduler(st)
}

// getOutboundIP 获得对外发送消息的 IP 地址
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
