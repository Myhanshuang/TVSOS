import request from '@/utils/request'
export const getPOIList = (params = {}) => {
  return request.get('/poi', { params })
} 