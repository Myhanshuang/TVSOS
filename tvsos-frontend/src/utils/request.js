import axios from 'axios'
import { useUserStore } from '@/stores';
import { ElMessage } from 'element-plus';
import router from '@/router';
const baseURL = '/api'
const instance = axios.create({
  baseURL,
  timeout: 10000
});

instance.interceptors.request.use(function (config) {
  const useStore = useUserStore()
  if (useStore.token) {
    config.headers.Authorization = `Bearer ${useStore.token}`
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
    if (error.response && error.response.status === 401) {
      // 未登录/过期：清 token 并回登录页
      const useStore = useUserStore()
      useStore.removeToken()
      router.push('/login')
      ElMessage.error('请先登录')
    } else {
      ElMessage.error((error.response && error.response.data && error.response.data.message) || '服务异常')
    }
    return Promise.reject(error);
  });
export default instance
export { baseURL }
