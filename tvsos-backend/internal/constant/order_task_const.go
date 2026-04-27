package constant

const (
	// 任务创建间隔时间
	OrderTaskCreatingGap = 20 // 20s执行一次

	// 任务阶段
	OrderTaskSequentialPlanned      = 1 // 已分配待发车
	OrderTaskSequentialAccepting    = 2 // 出发接单阶段
	OrderTaskSequentialTransporting = 3 // 运货送往目的地阶段
	OrderTaskSequentialFinish       = 4 // 任务完成
)
