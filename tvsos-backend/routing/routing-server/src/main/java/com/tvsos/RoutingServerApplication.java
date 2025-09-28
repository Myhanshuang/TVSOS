package com.tvsos;

import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@Slf4j
@SpringBootApplication
public class RoutingServerApplication {

    public static void main(String[] args) {
        SpringApplication.run(RoutingServerApplication.class, args);
        log.info("Routing Server 启动成功...");
    }

}
