package ioc

import (
	"github.com/alehua/cron-center/internal/pkg/logger"
	"go.uber.org/zap"
)

func InitLogger() logger.Logger {
	return logger.NewZapLogger(zap.L())
}
