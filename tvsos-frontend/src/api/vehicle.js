import request from '@/utils/request'

/**
 * 获取车辆列表数据
 * 根据前端传入的查询条件（如 status 等）向后端请求车辆的概要信息列表
 * @param {Object} params - 查询参数字典
 * @returns {Promise}
 */
export function getVehiclesData(params) {
  return request({
    url: '/vehicles',
    method: 'get',
    params
  })
}

/**
 * 获取指定车辆的历史/当前规划路径
 * 辅助大屏地图或组件，根据车辆 ID 拉取路线坐标点列
 * @param {String|Number} id - 目标车辆的唯一标识符
 * @returns {Promise}
 */
export function getVehiclePath(id) {
  return request({
    url: `/vehicles/path/${id}`,
    method: 'get'
  })
}
