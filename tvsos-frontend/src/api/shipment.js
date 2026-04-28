import request from '@/utils/request'

/**
 * 获取系统运单数据列表
 * 支持按运单 ID (num) 或 状态 (status) 进行筛选查询
 * @param {Object} params - 结构化参数对象，常包含 num 和 status
 * @returns {Promise}
 */
export const getShipmentList = ({ num, status } = {}) => {
  return request.get('/shipments', { params: { num, status } })
} 