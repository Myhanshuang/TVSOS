import request from '@/utils/request'

/**
 * 获取兴趣点 (POI) 列表数据
 * @param {Object} params - 查询参数 (如名称模糊查询、状态等)
 * @returns {Promise} 返回包含 POI 列表的 Promise 对象
 */
export const getPOIList = (params = {}) => {
  return request.get('/pois', { params })
} 