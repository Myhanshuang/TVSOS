package config

import (
	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var VP *viper.Viper

// InitViper 初始化 viper 并读取和加载配置文件
func InitViper() {
	// viper 定位配置文件
	Logger.Logger.Info("viper初始化配置文件...")
	VP = viper.New()
	VP.SetConfigName("config-dev")
	VP.SetConfigType("yml")
	VP.AddConfigPath(".")
	VP.AddConfigPath("../../") // 为测试增加路径支持

	// viper 读取并加载配置文件
	err := VP.ReadInConfig()
	if err != nil {
		Logger.Logger.Error("viper读取并加载配置文件失败", zap.Error(err))
		panic(err)
	}
	Logger.Logger.Info("viper读取并加载配置文件成功")
}
