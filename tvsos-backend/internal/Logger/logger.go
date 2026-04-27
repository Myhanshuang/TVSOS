package Logger

import "go.uber.org/zap"

var Logger *zap.Logger

// InitLogger 初始化全局 Logger 对象
func InitLogger() {
	var err error
	Logger, err = zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}
