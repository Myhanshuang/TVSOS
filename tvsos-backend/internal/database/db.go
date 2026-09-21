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

type tableMigration struct {
	Name string
	DDL  string
}

// DB 数据库连接对象
var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() {
	Logger.Logger.Info("初始化数据库连接...")
	dbname := config.VP.GetString("database.dbname")
	ip := config.VP.GetString("database.host")
	port := config.VP.GetString("database.port")
	username := config.VP.GetString("database.username")
	password := config.VP.GetString("database.password")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, ip, port, dbname)

	var err error
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
		{Name: "driver_id", DDL: "ALTER TABLE vehicle ADD COLUMN driver_id bigint unsigned NOT NULL DEFAULT 0 COMMENT '绑定司机id'"},
		{Name: "pause_until", DDL: "ALTER TABLE vehicle ADD COLUMN pause_until datetime DEFAULT NULL COMMENT '暂停截止时刻(休息/天气/维修统一暂停)'"},
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

	// 指标体系相关新表：指标采样 / 系统事件 / 排放因子 / 装卸参数 / 匹配日志 / 轨迹日志
	tableMigrations := []tableMigration{
		{
			Name: "metric_snapshot",
			DDL: `CREATE TABLE IF NOT EXISTS metric_snapshot (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				snapshot_time datetime NOT NULL,
				cost_value double NOT NULL DEFAULT 0 COMMENT '综合成本F',
				total_throughput double NOT NULL DEFAULT 0 COMMENT '总吞吐量(吨)',
				cargo_pending double NOT NULL DEFAULT 0 COMMENT '待处理货物(吨)',
				transport_efficiency double NOT NULL DEFAULT 0 COMMENT '运输效率(载重利用率)',
				total_waiting_time double NOT NULL DEFAULT 0 COMMENT '车辆总等待(秒)',
				total_empty_mileage double NOT NULL DEFAULT 0 COMMENT '总空驶里程(米)',
				unit_cost double NOT NULL DEFAULT 0 COMMENT '单位成本(成本/吨公里)',
				empty_rate double NOT NULL DEFAULT 0 COMMENT '空驶率(空驶/总里程)',
				loading_rate_stddev double NOT NULL DEFAULT 0 COMMENT '装载率标准差',
				match_response_p50 double NOT NULL DEFAULT 0 COMMENT '匹配响应P50(秒)',
				match_response_p95 double NOT NULL DEFAULT 0 COMMENT '匹配响应P95(秒)',
				match_response_p99 double NOT NULL DEFAULT 0 COMMENT '匹配响应P99(秒)',
				gini_coefficient double NOT NULL DEFAULT 0 COMMENT '基尼系数',
				carbon_total double NOT NULL DEFAULT 0 COMMENT '总碳排放(kg)',
				carbon_per_ton_km double NOT NULL DEFAULT 0 COMMENT '单位吨公里碳排放',
				PRIMARY KEY (id),
				KEY idx_snapshot_time (snapshot_time)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "system_event",
			DDL: `CREATE TABLE IF NOT EXISTS system_event (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				event_time datetime NOT NULL,
				event_type varchar(32) NOT NULL DEFAULT '' COMMENT '事件类型: startup/recovery/fault/recover',
				level varchar(16) NOT NULL DEFAULT 'info' COMMENT '级别',
				message varchar(512) NOT NULL DEFAULT '' COMMENT '描述',
				detail text COMMENT '详情(JSON)',
				PRIMARY KEY (id),
				KEY idx_event_time (event_time)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "emission_factor",
			DDL: `CREATE TABLE IF NOT EXISTS emission_factor (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				vehicle_type int NOT NULL COMMENT '车辆类型(对应vehicle.tybe)',
				factor_empty double NOT NULL DEFAULT 0 COMMENT '空驶排放因子(kg/km)',
				factor_load double NOT NULL DEFAULT 0 COMMENT '载重排放因子(kg/km)',
				remark varchar(255) NOT NULL DEFAULT '' COMMENT '备注',
				PRIMARY KEY (id),
				UNIQUE KEY uk_vehicle_type (vehicle_type)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "handling_config",
			DDL: `CREATE TABLE IF NOT EXISTS handling_config (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				config_key varchar(64) NOT NULL COMMENT '配置键',
				config_value double NOT NULL DEFAULT 0 COMMENT '配置值',
				remark varchar(255) NOT NULL DEFAULT '' COMMENT '说明',
				PRIMARY KEY (id),
				UNIQUE KEY uk_config_key (config_key)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "matching_log",
			DDL: `CREATE TABLE IF NOT EXISTS matching_log (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				shipment_id bigint unsigned NOT NULL COMMENT '运单id',
				chosen_vehicle_id bigint unsigned NOT NULL DEFAULT 0 COMMENT '选中车辆id',
				candidate_ranking text COMMENT '候选车辆排名(JSON)',
				ground_truth_vehicle bigint unsigned NOT NULL DEFAULT 0 COMMENT '最优车辆(评估用)',
				created_at datetime NOT NULL,
				PRIMARY KEY (id),
				KEY idx_shipment (shipment_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "driver",
			DDL: `CREATE TABLE IF NOT EXISTS driver (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				name varchar(64) NOT NULL DEFAULT '' COMMENT '姓名',
				phone varchar(32) NOT NULL DEFAULT '' COMMENT '电话',
				license_no varchar(64) NOT NULL DEFAULT '' COMMENT '驾驶证号',
				level int NOT NULL DEFAULT 3 COMMENT '驾驶等级 1-5',
				fatigue double NOT NULL DEFAULT 0 COMMENT '疲劳度 0~1',
				continuous_hours double NOT NULL DEFAULT 0 COMMENT '连续驾驶小时数',
				rest_until datetime DEFAULT NULL COMMENT '休息截止时刻',
				status int NOT NULL DEFAULT 1 COMMENT '1可用/2休息中',
				refuse_count int NOT NULL DEFAULT 0 COMMENT '累计拒单次数',
				vehicle_id bigint unsigned NOT NULL DEFAULT 0 COMMENT '绑定车辆id',
				update_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				PRIMARY KEY (id),
				KEY idx_vehicle (vehicle_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "random_event",
			DDL: `CREATE TABLE IF NOT EXISTS random_event (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				event_time datetime NOT NULL,
				event_type varchar(32) NOT NULL DEFAULT '' COMMENT 'refuse/rest/weather/accident',
				vehicle_id bigint unsigned NOT NULL DEFAULT 0 COMMENT '车辆id',
				driver_id bigint unsigned NOT NULL DEFAULT 0 COMMENT '司机id',
				shipment_id bigint unsigned NOT NULL DEFAULT 0 COMMENT '运单id',
				task_id bigint unsigned NOT NULL DEFAULT 0 COMMENT '任务id',
				severity int NOT NULL DEFAULT 1 COMMENT '严重程度 1-5',
				duration_seconds int NOT NULL DEFAULT 0 COMMENT '影响时长(秒)',
				description varchar(512) NOT NULL DEFAULT '' COMMENT '描述',
				detail text COMMENT '详情(JSON)',
				status varchar(16) NOT NULL DEFAULT 'active' COMMENT 'active/pending/resolved',
				PRIMARY KEY (id),
				KEY idx_event_time (event_time),
				KEY idx_vehicle (vehicle_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "track_log",
			DDL: `CREATE TABLE IF NOT EXISTS track_log (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				vehicle_id bigint unsigned NOT NULL COMMENT '车辆id',
				lon double NOT NULL DEFAULT 0 COMMENT '经度',
				lat double NOT NULL DEFAULT 0 COMMENT '纬度',
				is_empty tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否空驶段',
				record_time datetime NOT NULL,
				PRIMARY KEY (id),
				KEY idx_vehicle_time (vehicle_id, record_time)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
	}

	for _, tm := range tableMigrations {
		var tableCount int64
		err = DB.Raw(`
			SELECT COUNT(*)
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = ?
		`, tm.Name).Scan(&tableCount).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：检查表", zap.String("table", tm.Name), zap.Error(err))
			panic(err)
		}
		if tableCount > 0 {
			continue
		}
		err = DB.Exec(tm.DDL).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：创建表", zap.String("table", tm.Name), zap.Error(err))
			panic(err)
		}
		Logger.Logger.Info("数据库迁移：已创建表", zap.String("table", tm.Name))
	}

	// 沙箱系统元数据表：用户(研究人员) / 沙箱 / 实验方案 / 实验(批次) / 实验终态结果
	sandboxTableMigrations := []tableMigration{
		{
			Name: "user",
			DDL: `CREATE TABLE IF NOT EXISTS user (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				username varchar(64) NOT NULL COMMENT '登录名',
				password_hash varchar(128) NOT NULL DEFAULT '' COMMENT 'bcrypt密码哈希',
				display_name varchar(64) NOT NULL DEFAULT '' COMMENT '显示名(研究人员姓名)',
				created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (id),
				UNIQUE KEY uk_username (username)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "sandbox",
			DDL: `CREATE TABLE IF NOT EXISTS sandbox (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				name varchar(128) NOT NULL DEFAULT '' COMMENT '沙箱名',
				description varchar(1024) NOT NULL DEFAULT '' COMMENT '描述',
				world_rules_toml mediumtext COMMENT '世界规则TOML全文(geography/traffic/time/economy/world.*)',
				fleet_template mediumtext COMMENT '车队模板JSON(车辆类型库/车队定义)',
				poi_whitelist text COMMENT 'POI白名单JSON(空=全部)',
				algo_version varchar(64) NOT NULL DEFAULT '' COMMENT '绑定调度算法版本',
				created_by bigint unsigned NOT NULL DEFAULT 0 COMMENT '创建研究者user.id',
				created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
				archived tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否归档',
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "experiment_scheme",
			DDL: `CREATE TABLE IF NOT EXISTS experiment_scheme (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				sandbox_id bigint unsigned NOT NULL COMMENT '所属沙箱',
				name varchar(128) NOT NULL DEFAULT '' COMMENT '方案名',
				description varchar(1024) NOT NULL DEFAULT '' COMMENT '描述',
				params_toml mediumtext COMMENT '实验级参数TOML模板',
				created_by bigint unsigned NOT NULL DEFAULT 0 COMMENT '创建研究者user.id',
				created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
				update_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				PRIMARY KEY (id),
				KEY idx_sandbox (sandbox_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "experiment",
			DDL: `CREATE TABLE IF NOT EXISTS experiment (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				sandbox_id bigint unsigned NOT NULL COMMENT '所属沙箱',
				scheme_id bigint unsigned NOT NULL DEFAULT 0 COMMENT '来源方案(0=临时实验)',
				name varchar(128) NOT NULL DEFAULT '' COMMENT '实验名',
				params_toml mediumtext COMMENT '参数快照TOML',
				seed bigint NOT NULL DEFAULT 0 COMMENT '随机种子',
				status varchar(16) NOT NULL DEFAULT 'configured' COMMENT 'configured/generating/ready/running/finished/failed/aborted',
				data_version int NOT NULL DEFAULT 0 COMMENT '数据生成器版本(种子导入校验)',
				progress double NOT NULL DEFAULT 0 COMMENT '虚拟时钟进度0~1',
				created_by bigint unsigned NOT NULL DEFAULT 0 COMMENT '实验者user.id',
				created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
				started_at datetime DEFAULT NULL,
				finished_at datetime DEFAULT NULL,
				PRIMARY KEY (id),
				KEY idx_sandbox (sandbox_id),
				KEY idx_status (status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "experiment_result",
			DDL: `CREATE TABLE IF NOT EXISTS experiment_result (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				experiment_id bigint unsigned NOT NULL COMMENT '所属实验',
				summary_json mediumtext COMMENT '终态指标全量快照(JSON)',
				primary_metrics text COMMENT '主目标C2/U2/U4/T2/F1/G2等(JSON)',
				convergence mediumtext COMMENT 'SA收敛曲线/算法性能(JSON)',
				created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (id),
				UNIQUE KEY uk_experiment (experiment_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
	}

	for _, tm := range sandboxTableMigrations {
		var tableCount int64
		err = DB.Raw(`
			SELECT COUNT(*)
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = ?
		`, tm.Name).Scan(&tableCount).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：检查沙箱表", zap.String("table", tm.Name), zap.Error(err))
			panic(err)
		}
		if tableCount > 0 {
			continue
		}
		err = DB.Exec(tm.DDL).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：创建沙箱表", zap.String("table", tm.Name), zap.Error(err))
			panic(err)
		}
		Logger.Logger.Info("数据库迁移：已创建沙箱表", zap.String("table", tm.Name))
	}

	// experiment 司机等级表列迁移（数据生成期种子摇号，实例化与脚本事件共用）
	{
		var columnCount int64
		err = DB.Raw(`
			SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'experiment' AND COLUMN_NAME = 'driver_levels'
		`).Scan(&columnCount).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：检查 experiment.driver_levels", zap.Error(err))
			panic(err)
		}
		if columnCount == 0 {
			if err = DB.Exec("ALTER TABLE experiment ADD COLUMN driver_levels text COMMENT '司机等级表JSON(车牌→等级)'").Error; err != nil {
				Logger.Logger.Error("数据库迁移失败：新增 experiment.driver_levels", zap.Error(err))
				panic(err)
			}
			Logger.Logger.Info("数据库迁移：已新增 experiment.driver_levels")
		}
	}

	// 运行数据表实验维度列迁移：experiment_id=0 表示沙箱模板/全局数据，>0 表示某次实验的实例数据。
	experimentScopedTables := []string{
		"vehicle", "driver", "shipment", "order_task",
		"random_event", "matching_log", "metric_snapshot", "track_log", "system_event",
	}
	for _, table := range experimentScopedTables {
		var columnCount int64
		err = DB.Raw(`
			SELECT COUNT(*)
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = ?
			  AND COLUMN_NAME = 'experiment_id'
		`, table).Scan(&columnCount).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：检查实验维度列", zap.String("table", table), zap.Error(err))
			panic(err)
		}
		if columnCount > 0 {
			continue
		}
		err = DB.Exec("ALTER TABLE " + table + " ADD COLUMN experiment_id bigint unsigned NOT NULL DEFAULT 0 COMMENT '所属实验id(0=沙箱模板/全局)', ADD INDEX idx_experiment (experiment_id)").Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：新增实验维度列", zap.String("table", table), zap.Error(err))
			panic(err)
		}
		Logger.Logger.Info("数据库迁移：已新增实验维度列", zap.String("table", table))
	}

	// 车辆载货时长列迁移：loaded_duration 累计运输(重载)段行驶小时，用于Top载货时长榜。
	{
		var columnCount int64
		err = DB.Raw(`
			SELECT COUNT(*)
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = 'vehicle'
			  AND COLUMN_NAME = 'loaded_duration'
		`).Scan(&columnCount).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：检查 vehicle.loaded_duration", zap.Error(err))
			panic(err)
		}
		if columnCount == 0 {
			if err = DB.Exec("ALTER TABLE vehicle ADD COLUMN loaded_duration double NOT NULL DEFAULT 0 COMMENT '累计载货时长(小时)'").Error; err != nil {
				Logger.Logger.Error("数据库迁移失败：新增 vehicle.loaded_duration", zap.Error(err))
				panic(err)
			}
			Logger.Logger.Info("数据库迁移：已新增 vehicle.loaded_duration")
		}
	}

	// 沙箱预生成脚本表：实验订单流 / 随机事件脚本（数据生成器输出）
	scriptTableMigrations := []tableMigration{
		{
			Name: "experiment_order",
			DDL: `CREATE TABLE IF NOT EXISTS experiment_order (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				experiment_id bigint unsigned NOT NULL COMMENT '所属实验',
				virtual_time datetime NOT NULL COMMENT '虚拟释放时刻',
				start_poi_id bigint unsigned NOT NULL DEFAULT 0,
				end_poi_id bigint unsigned NOT NULL DEFAULT 0,
				cargo_id bigint unsigned NOT NULL DEFAULT 0,
				count int NOT NULL DEFAULT 1,
				released tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否已释放为shipment',
				PRIMARY KEY (id),
				KEY idx_exp_time (experiment_id, virtual_time)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
		{
			Name: "experiment_event",
			DDL: `CREATE TABLE IF NOT EXISTS experiment_event (
				id bigint unsigned NOT NULL AUTO_INCREMENT,
				experiment_id bigint unsigned NOT NULL COMMENT '所属实验',
				virtual_time datetime NOT NULL COMMENT '虚拟触发时刻',
				event_type varchar(32) NOT NULL DEFAULT '' COMMENT 'weather/accident',
				vehicle_license varchar(64) NOT NULL DEFAULT '' COMMENT '目标车辆车牌(空=全体)',
				severity int NOT NULL DEFAULT 1,
				duration_seconds int NOT NULL DEFAULT 0 COMMENT '影响时长(虚拟秒)',
				detail text,
				consumed tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否已消费',
				PRIMARY KEY (id),
				KEY idx_exp_time (experiment_id, virtual_time)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		},
	}
	for _, tm := range scriptTableMigrations {
		var tableCount int64
		err = DB.Raw(`
			SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		`, tm.Name).Scan(&tableCount).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：检查脚本表", zap.String("table", tm.Name), zap.Error(err))
			panic(err)
		}
		if tableCount > 0 {
			continue
		}
		if err = DB.Exec(tm.DDL).Error; err != nil {
			Logger.Logger.Error("数据库迁移失败：创建脚本表", zap.String("table", tm.Name), zap.Error(err))
			panic(err)
		}
		Logger.Logger.Info("数据库迁移：已创建脚本表", zap.String("table", tm.Name))
	}

	// order_task 任务阶段时间戳列迁移（支撑调度时序还原与响应时间分位统计）
	orderTaskColumnMigrations := []vehicleColumnMigration{
		{Name: "assign_time", DDL: "ALTER TABLE order_task ADD COLUMN assign_time datetime DEFAULT NULL COMMENT '订单被分配车辆时间'"},
		{Name: "reach_pickup_time", DDL: "ALTER TABLE order_task ADD COLUMN reach_pickup_time datetime DEFAULT NULL COMMENT '车辆到达提货点时间'"},
		{Name: "load_start_time", DDL: "ALTER TABLE order_task ADD COLUMN load_start_time datetime DEFAULT NULL COMMENT '开始装货时间'"},
		{Name: "load_end_time", DDL: "ALTER TABLE order_task ADD COLUMN load_end_time datetime DEFAULT NULL COMMENT '完成装货时间'"},
		{Name: "transport_start_time", DDL: "ALTER TABLE order_task ADD COLUMN transport_start_time datetime DEFAULT NULL COMMENT '开始运输时间'"},
		{Name: "reach_delivery_time", DDL: "ALTER TABLE order_task ADD COLUMN reach_delivery_time datetime DEFAULT NULL COMMENT '到达收货点时间'"},
		{Name: "unload_start_time", DDL: "ALTER TABLE order_task ADD COLUMN unload_start_time datetime DEFAULT NULL COMMENT '开始卸货时间'"},
		{Name: "unload_end_time", DDL: "ALTER TABLE order_task ADD COLUMN unload_end_time datetime DEFAULT NULL COMMENT '完成卸货时间'"},
		{Name: "signoff_time", DDL: "ALTER TABLE order_task ADD COLUMN signoff_time datetime DEFAULT NULL COMMENT '签收完成时间'"},
	}

	for _, migration := range orderTaskColumnMigrations {
		var columnCount int64
		err = DB.Raw(`
			SELECT COUNT(*)
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = 'order_task'
			  AND COLUMN_NAME = ?
		`, migration.Name).Scan(&columnCount).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：检查 order_task 字段", zap.String("column", migration.Name), zap.Error(err))
			panic(err)
		}
		if columnCount > 0 {
			continue
		}
		err = DB.Exec(migration.DDL).Error
		if err != nil {
			Logger.Logger.Error("数据库迁移失败：新增 order_task 字段", zap.String("column", migration.Name), zap.Error(err))
			panic(err)
		}
	}

	Logger.Logger.Info("数据库连接成功")
}