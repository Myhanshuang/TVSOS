package com.example.gpsbackend.gps;

import lombok.Data;

/**
 * 用于接收从Python脚本发送的GPS数据的DTO。
 */
@Data
public class GpsData {
    private Double latitude;
    private Double longitude;
    private Double speed;
    private String device_id;
    private Double timestamp; // Python的time.time()返回浮点数，因此这里使用Double
}