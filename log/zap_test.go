package log

import "testing"

func TestInit(t *testing.T) {
	zf := Config{
		// LogFile:    "zap.log", // 不传就不写文件
		Level:      0,
		MaxAge:     1,
		MaxSize:    1,
		MaxBackups: 1,
		Compress:   true,
		JsonFormat: false,
	}
	Init(zf)
	Sugar.Info("zap log", "success", true, 1)
	Sugar.Infof("zap log success %t %d", true, 1)
	Sugar.Infow("zap log", "success", true)
}
