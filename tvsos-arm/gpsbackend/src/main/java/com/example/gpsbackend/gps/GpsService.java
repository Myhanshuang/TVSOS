package com.example.gpsbackend.gps;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;
import org.springframework.http.ResponseEntity;
import java.util.concurrent.atomic.AtomicBoolean; // 新增导入

@Service
public class GpsService {

    // Python连接状态标志
    private final AtomicBoolean pythonConnected = new AtomicBoolean(false);

    @Value("${python.flask.url}")
    private String pythonFlaskUrl;
    private final RestTemplate restTemplate;

    public GpsService() {
        this.restTemplate = new RestTemplate();
    }

    // 设置连接状态的方法
    public void setPythonConnected(boolean connected) {
        pythonConnected.set(connected);
        System.out.println("Python连接状态更新: " + (connected ? "已连接" : "未连接"));
    }

    @Scheduled(initialDelay = 5000, fixedRate = 5000)
    public void sendRequestToPythonPeriodically() {
        // 检查连接状态
        if (!pythonConnected.get()) {
            System.out.println("定时任务跳过：Python尚未建立连接");
            return;
        }

        System.out.println("定时任务触发：准备向Python请求GPS数据...");
        triggerPythonGpsCollection();
    }

    public void triggerPythonGpsCollection() {
        String url = pythonFlaskUrl + "/notify_request_received";
        System.out.println("正在发送请求到Python Flask应用:" + url);
        try {
            ResponseEntity<String> response = restTemplate.postForEntity(url, null, String.class);
            System.out.println("已向Python发送GPS数据收集请求。响应状态码: " + response.getStatusCode());
            System.out.println("响应体:" + response.getBody());
        } catch (Exception e) {
            System.err.println("向Python发送GPS数据收集请求失败:" + e.getMessage());

            // 请求失败时重置连接状态
            setPythonConnected(false);
        }
    }

    // 在GpsService类中添加以下方法
    @Scheduled(fixedRate = 30000) // 每30秒检查一次
    public void checkConnectionTimeout() {
        if (pythonConnected.get()) {
            System.out.println("连接状态检查：Python保持连接");
            return;
        }

        // 尝试重新建立连接
        System.out.println("尝试重新连接Python服务...");
        try {
            String testUrl = pythonFlaskUrl.replace("/notify_request_received", "");
            restTemplate.getForEntity(testUrl, String.class);
            setPythonConnected(true);
        } catch (Exception e) {
            System.err.println("Python服务不可用: " + e.getMessage());
        }
    }
}