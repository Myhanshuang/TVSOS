import request from '@/utils/request'
export const getVehicleCategory = (params = {}) => {
  return request.get('/report/vehicleCategory', { params })
} 

export const getPoiTybe = (params = {}) => {
  return request.get('/report/poiTybe', { params })
} 

// 统计车辆数量
export const getVehicleSum = (params = {}) => {
  return request.get('/report/vehicleSum', { params })
} 

// 统计 poi 数量
export const getPoiSum = (params = {}) => {
  return request.get('/report/poiSum', { params })
} 

// 统计司机数量
export const getDriverSum = (params = {}) => {
  return request.get('/report/driverSum', { params })
} 

// 统计 车上的货物总量 /kg
export const getCargoSizeSum = (params = {}) => {
  return request.get('/report/cargoSize', { params })
} 
