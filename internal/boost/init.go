package boost

import (
	"context"
	"github.com/alehua/cron-center/internal/schedule"
	"github.com/alehua/cron-center/internal/storage"
	"log"
	"net"
)

func WithStart(ctx context.Context) error {
	l := InitLogger()
	db := InitDB(l)
	instantId := getOutboundIP() // 计算本级IP作为实例的ID
	st := storage.NewTaskStorage(db, instantId,
		storage.WithRefreshLimit(10))

	sche := schedule.NewScheduler(st)

	num := register(ctx, sche)
	log.Printf("添加任务个数=%d\n", num)

	// 启动抢占和续约
	go st.Preempt(ctx)
	go st.AutoRefresh(ctx)

	return sche.Start(ctx)
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
