package com.example.gpsbackend;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling; 

@SpringBootApplication
@EnableScheduling // 启用Spring Boot的定时任务调度功能

public class GpsbackendApplication {
    public static void main(String[] args) {
        SpringApplication.run(GpsbackendApplication.class, args);
    }
}