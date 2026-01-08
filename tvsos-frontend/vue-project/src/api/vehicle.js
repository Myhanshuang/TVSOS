import request from '@/utils/request'

export function getVehiclesData(params) {
  return request({
    url: '/vehicles',
    method: 'get',
    params
  })
}

export function getVehiclePath(id) {
  return request({
    url: `/vehicles/path/${id}`,
    method: 'get'
  })
}
