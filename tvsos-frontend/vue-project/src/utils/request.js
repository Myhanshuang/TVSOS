import axios from 'axios'
import { useUserStore } from '@/stores';
import { ElMessage } from 'element-plus';
const baseURL = '/api'
const instance = axios.create({
  baseURL,
  timeout: 10000
});

instance.interceptors.request.use(function (config) {
  const useStore = useUserStore()
  if (config.token) {
    config.headers.Authorization = useStore.token
  }
  return config;
}, function (error) {
  return Promise.reject(error);
});

instance.interceptors.response.use(function (response) {
  if (response.data.code == 1) {
    return response
  }
  ElMessage.error(response.data.message || '服务异常')
  return response;
},
  function (error) {
    ElMessage.error(error.response.data.message || '服务异常')
    return Promise.reject(error);
  });
export default instance
export { baseURL }