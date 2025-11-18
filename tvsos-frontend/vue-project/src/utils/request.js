import axios from 'axios'
import { useUserStore } from '@/stores';
import { ElMessage } from 'element-plus';
import router from '@/router';
const baseURL='/api'
const instance = axios.create({
  baseURL,
  timeout: 10000
});
// 添加请求拦截器
instance.interceptors.request.use(function (config) {
    // 在发送请求之前做些什么
    const useStore=useUserStore()
    if(config.token){
      config.headers.Authorization=useStore.token
    }
    return config;
  }, function (error) {
    // 对请求错误做些什么
    return Promise.reject(error);
  });

// 添加响应拦截器
instance.interceptors.response.use(function (response) {
    // 2xx 范围内的状态码都会触发该函数。
    // 对响应数据做点什么
    if(response.data.code==1){
      return response
    }
    ElMessage.error(response.data.message||'服务异常')
    return response;
  },
  function (error) {
    // 超出 2xx 范围的状态码都会触发该函数。
    // 对响应错误做点什么
    ElMessage.error(error.response.data.message||'服务异常')
    return Promise.reject(error);
  });
export default instance
export {baseURL}