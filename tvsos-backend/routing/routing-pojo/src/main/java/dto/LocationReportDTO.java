package dto;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

// 这个类专门用来接收硬件上报的JSON数据
@AllArgsConstructor
@NoArgsConstructor
@Data
public class LocationReportDTO {
    private String license;
    private Double lon;
    private Double lat;
    private Double speed;
}