import request from '@/utils/request'
export const userRegister=({email,password,repassword})=>{
    return request.post('/api/reg',{email,password,repassword})
}
export const userLogin=({email,password})=>
    request.post('api/login',{email,password})