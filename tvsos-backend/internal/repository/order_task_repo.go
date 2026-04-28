package repository

import (
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// CreateOrderTask 新增一条任务记录 (OrderTask)
// 当运单(Shipment)指派给某车并下达调度指令后，将在此生成具体物理承运任务的数据，并落库记录。
func CreateOrderTask(task *model.OrderTask) error {
	// 获取数据库连接对象
	db := database.DB.Model(&model.OrderTask{})
	// 插入数据
	err := db.Create(task).Error
	return err
}

// GetRunningTasks 获取当前所有处于活跃进行中的派单执行任务集。
// 会过滤出状态为 "接收调度去装货" 和 "运输前往终点" 的记录数组返回。
func GetRunningTasks() ([]*model.OrderTask, error) {
	// 存放查询结果
	var tasks []*model.OrderTask
	db := database.DB.Model(&model.OrderTask{})
	db = db.Where("sequential in (?, ?)", constant.OrderTaskSequentialAccepting, constant.OrderTaskSequentialTransporting)
	err := db.Find(&tasks).Error
	return tasks, err
}

// GetRunningTasksOrderedByVehicle 查询当前所有进行中的有效任务，并对它们按挂靠小车及分配创建时间升序排列。
// 调度评估预测装载分布和积压时间时，会依据队列先后推算预期积压。
func GetRunningTasksOrderedByVehicle() ([]*model.OrderTask, error) {
	var tasks []*model.OrderTask
	db := database.DB.Model(&model.OrderTask{})
	db = db.Where("sequential in (?, ?)", constant.OrderTaskSequentialAccepting, constant.OrderTaskSequentialTransporting)
	db = db.Order("vehicle_id, create_time ASC")
	err := db.Find(&tasks).Error
	return tasks, err
}

// GetAllTasks 平铺查询并返回全量历史（含废弃与成功）所有跑单执行数据记录。
// 一般用于生成完整的历史报表以及后端算法的深度统计归档。
func GetAllTasks() ([]*model.OrderTask, error) {
	var tasks []*model.OrderTask
	db := database.DB.Model(&model.OrderTask{})
	err := db.Find(&tasks).Error
	return tasks, err
}

// UpdateTaskSequential 基于乐观锁 (CAS) 在流转流程中推进该指派任务的调度阶段。
// oldStatus 要求匹配目标记录以防并发冲突更新，确保任务如期进入预期的新生命周期阶段。
func UpdateTaskSequential(id int, status int, oldStatus int) error {
	// 获取数据库连接对象
	db := database.DB.Model(&model.OrderTask{})
	// CAS锁
	db = db.Where("id = ? and sequential = ?", id, oldStatus)
	db = db.Update("sequential", status)
	return db.Error
}

// UpdateTaskStatus 直接（非安全）更新数据库里指定任务实体的当前状态位 (无 CAS 限制)。
func UpdateTaskStatus(id int, status int) error {
	db := database.DB.Model(&model.OrderTask{})
	db = db.Where("id = ?", id)
	err := db.Update("sequential", status).Error
	return err
}

// GetFirstTaskByVehicleAndSequential 按创建时间获取某车辆上处于特定流水线阶段的最早的一条派工任务。
// 通常用它寻找挂靠在车辆队列中的头部任务，用以触发后续业务逻辑。
func GetFirstTaskByVehicleAndSequential(vehicleID uint, sequential int) (*model.OrderTask, error) {
	var task model.OrderTask
	db := database.DB.Model(&model.OrderTask{})
	err := db.Where("vehicle_id = ? and sequential = ?", vehicleID, sequential).
		Order("create_time ASC").
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetFirstRunningTaskByVehicle 从车辆承接所有处于活动(装货去程或者运载前往终点)的指派作业栈中，提取最早建立的那笔实体。
func GetFirstRunningTaskByVehicle(vehicleID uint) (*model.OrderTask, error) {
	var task model.OrderTask
	db := database.DB.Model(&model.OrderTask{})
	err := db.Where("vehicle_id = ? and sequential in (?, ?)",
		vehicleID,
		constant.OrderTaskSequentialAccepting,
		constant.OrderTaskSequentialTransporting,
	).
		Order("create_time ASC").
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}
