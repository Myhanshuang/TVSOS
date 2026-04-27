package database

import (
	"fmt"

	"github.com/kiritosuki/mover/internal/Logger"
	"github.com/kiritosuki/mover/internal/config"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type vehicleColumnMigration struct {
	Name string
	DDL  string
}

// DB 数据库连接对象
var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() {
	Logger.Logger.Info("初始化数据库连接...")
	// 获取数据库名称
	dbname := config.VP.GetString("database.dbname")
	// 获取数据库运行 ip
	ip := config.VP.GetString("database.host")
	// 获取数据库运行端口
	port := config.VP.GetString("database.port")
	// 获取数据库用户名
	username := config.VP.GetString("database.username")
	// 获取数据库密码
	password := config.VP.GetString("database.password")

	// DataSourceName 数据库连接字符串
	// 格式：<username>:<password>@tcp(<ip>:<port>)/<数据库名>?<参数设置>=<...>
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, ip, port, dbname)

	// 创建数据库连接对象
	var err error
	// 第二个参数配置sql日志
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		Logger.Logger.Error("数据库连接失败！", zap.Error(err))
		panic(err)
	}

	// 兼容迁移：为车辆详情面板补齐字段。
	migrations := []vehicleColumnMigration{
		{Name: "reserved_size", DDL: "ALTER TABLE vehicle ADD COLUMN reserved_size int NOT NULL DEFAULT 0 COMMENT '预占载重(kg)'"},
		{Name: "license", DDL: "ALTER TABLE vehicle ADD COLUMN license varchar(64) NOT NULL DEFAULT '' COMMENT '车牌号'"},
		{Name: "length", DDL: "ALTER TABLE vehicle ADD COLUMN length double NOT NULL DEFAULT 0 COMMENT '车厢长度(米)'"},
		{Name: "width", DDL: "ALTER TABLE vehicle ADD COLUMN width double NOT NULL DEFAULT 0 COMMENT '车厢宽度(米)'"},
		{Name: "height", DDL: "ALTER TABLE vehicle ADD COLUMN height double NOT NULL DEFAULT 0 COMMENT '车厢高度(米)'"},
		{Name: "wait_time", DDL: "ALTER TABLE vehicle ADD COLUMN wait_time double NOT NULL DEFAULT 0 COMMENT '当前等待时长(分钟)'"},
		{Name: "total_wait_time", DDL: "ALTER TABLE vehicle ADD COLUMN total_wait_time double NOT NULL DEFAULT 0 COMMENT '累计等待时长(分钟)'"},
		{Name: "empty_mileage", DDL: "ALTER TABLE vehicle ADD COLUMN empty_mileage double NOT NULL DEFAULT 0 COMMENT '空驶里程(千米)'"},
		{Name: "distance", DDL: "ALTER TABLE vehicle ADD COLUMN distance double NOT NULL DEFAULT 0 COMMENT '运输里程(千米)'"},
		{Name: "duration", DDL: "ALTER TABLE vehicle ADD COLUMN duration double NOT NULL DEFAULT 0 COMMENT '运输时长(小时)'"},
		{Name: "angle", DDL: "ALTER TABLE vehicle ADD COLUMN angle double NOT NULL DEFAULT 90 COMMENT '车辆角度(度)'"},
	}

	for _, migration := range migrations {
		var columnCount int64
		err = DB.Raw(`
			SELECT COUNT(*)
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = 'vehicle'
			  AND COLUMN_NAME = ?
		`, migration.Name).Scan(&columnCount).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：检查 vehicle 字段", zap.String("column", migration.Name), zap.Error(err))
			panic(err)
		}

		if columnCount > 0 {
			continue
		}

		err = DB.Exec(migration.DDL).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：新增 vehicle 字段", zap.String("column", migration.Name), zap.Error(err))
			panic(err)
		}
	}

	Logger.Logger.Info("数据库连接成功")
}
