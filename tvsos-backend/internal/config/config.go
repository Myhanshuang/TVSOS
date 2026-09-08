package config

import (
	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var VP *viper.Viper

// InitViper 初始化 viper 并读取和加载配置文件
func InitViper() {
	if err := InitViperWithName("config-dev"); err != nil {
		Logger.Logger.Error("viper读取并加载配置文件失败", zap.Error(err))
		panic(err)
	}
}

// InitViperWithName 以指定配置名初始化 viper（测试专用：config-test.yml 指向独立测试库）。
// 与 InitViper 不同，读取失败时返回 error 而不是 panic，便于测试 Skip。
func InitViperWithName(name string) error {
	Logger.Logger.Info("viper初始化配置文件...")
	VP = viper.New()
	VP.SetConfigName(name)
	VP.SetConfigType("yml")
	VP.AddConfigPath(".")
	VP.AddConfigPath("../../") // 为测试增加路径支持

	if err := VP.ReadInConfig(); err != nil {
		return err
	}
	Logger.Logger.Info("viper读取并加载配置文件成功")
	return nil
}
