// src/api/vehicle.js
import request from '@/utils/request';

export const getVehiclesCurrentData = () => {
    return request.get('/vehicles/current_data'); //后续替换为真实地址
};