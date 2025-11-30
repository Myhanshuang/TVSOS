// src/api/vehicle.js
import request from '@/utils/request';

export const getVehiclesData = () => {
    return request.get('/vehicles'); 
};