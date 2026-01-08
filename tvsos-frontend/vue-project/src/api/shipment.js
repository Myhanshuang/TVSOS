import request from '@/utils/request'
export const getShipmentList = ({ num, status } = {}) => {
  return request.get('/shipments', { params: { num, status } })
} 