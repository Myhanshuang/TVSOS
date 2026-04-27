package repository

import (
	"github.com/kiritosuki/mover/internal/constant"
	"github.com/kiritosuki/mover/internal/database"
	"github.com/kiritosuki/mover/internal/model"
)

// CreateOrderTask 插入一条任务
func CreateOrderTask(task *model.OrderTask) error {
	// 获取数据库连接对象
	db := database.DB.Model(&model.OrderTask{})
	// 插入数据
	err := db.Create(task).Error
	return err
}

// GetRunningTasks 查询正在进行中的任务
func GetRunningTasks() ([]*model.OrderTask, error) {
	// 存放查询结果
	var tasks []*model.OrderTask
	db := database.DB.Model(&model.OrderTask{})
	db = db.Where("sequential in (?, ?)", constant.OrderTaskSequentialAccepting, constant.OrderTaskSequentialTransporting)
	err := db.Find(&tasks).Error
	return tasks, err
}

// GetRunningTasksOrderedByVehicle 查询正在进行中的任务，按车辆和时间排序
func GetRunningTasksOrderedByVehicle() ([]*model.OrderTask, error) {
	var tasks []*model.OrderTask
	db := database.DB.Model(&model.OrderTask{})
	db = db.Where("sequential in (?, ?)", constant.OrderTaskSequentialAccepting, constant.OrderTaskSequentialTransporting)
	db = db.Order("vehicle_id, create_time ASC")
	err := db.Find(&tasks).Error
	return tasks, err
}

// GetAllTasks 查询所有任务（用于统计）
func GetAllTasks() ([]*model.OrderTask, error) {
	var tasks []*model.OrderTask
	db := database.DB.Model(&model.OrderTask{})
	err := db.Find(&tasks).Error
	return tasks, err
}

// UpdateTaskSequential 更新任务进入下一阶段
func UpdateTaskSequential(id int, status int, oldStatus int) error {
	// 获取数据库连接对象
	db := database.DB.Model(&model.OrderTask{})
	// CAS锁
	db = db.Where("id = ? and sequential = ?", id, oldStatus)
	db = db.Update("sequential", status)
	return db.Error
}

// UpdateTaskStatus 更新任务状态
func UpdateTaskStatus(id int, status int) error {
	db := database.DB.Model(&model.OrderTask{})
	db = db.Where("id = ?", id)
	err := db.Update("sequential", status).Error
	return err
}

// GetFirstTaskByVehicleAndSequential 查询某辆车按创建时间最早的一条指定阶段任务
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

// GetFirstRunningTaskByVehicle 查询某辆车当前正在执行中的最早任务
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
