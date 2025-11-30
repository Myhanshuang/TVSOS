package com.tvsos;

import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

@Slf4j
@SpringBootApplication
@EnableScheduling
public class RoutingServerApplication {

    public static void main(String[] args) {
        SpringApplication.run(RoutingServerApplication.class, args);
        log.info("Routing Server 启动成功...");
    }

}
