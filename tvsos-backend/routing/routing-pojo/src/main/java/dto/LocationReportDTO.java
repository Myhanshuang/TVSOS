package dto;

import lombok.Data;

// 这个类专门用来接收硬件上报的JSON数据
@Data
public class LocationReportDTO {
    private String license;
    private Double lon;
    private Double lat;
    private Double speed;
    private Long timestamp;
}