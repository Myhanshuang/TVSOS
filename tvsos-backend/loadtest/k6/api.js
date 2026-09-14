import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const baseURL = __ENV.BASE_URL || 'http://127.0.0.1:8088';
const endpoint = __ENV.ENDPOINT || '/vehicles';
const businessFailures = new Rate('business_failures');

export const options = {
  vus: Number(__ENV.VUS || 10),
  duration: __ENV.DURATION || '15s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    business_failures: ['rate<0.01'],
    http_req_duration: ['p(95)<1500'],
  },
};

export default function () {
  const response = http.get(`${baseURL}${endpoint}`, {
    tags: { endpoint },
  });

  let businessOK = false;
  try {
    businessOK = response.json('code') === 1;
  } catch (_) {
    businessOK = false;
  }
  businessFailures.add(!businessOK);

  check(response, {
    'HTTP 状态为 200': (r) => r.status === 200,
    '业务状态成功': () => businessOK,
  });

  // 模拟用户连续查看看板时的短暂停顿，避免单机测试先耗尽客户端临时端口。
  sleep(Number(__ENV.THINK_TIME || 0.1));
}
