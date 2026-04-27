import request from '@/utils/request'

export const setSimulationSpeed = (multiplier) => {
  return request({
    url: '/simulation/speed',
    method: 'post',
    params: { multiplier }
  })
}

export const getSimulationSpeed = () => {
  return request({
    url: '/simulation/speed',
    method: 'get'
  })
}
