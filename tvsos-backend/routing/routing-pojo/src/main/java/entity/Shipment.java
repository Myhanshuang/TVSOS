package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Shipment {
    private Long id;
    private String num;
    private Double beginLon;
    private Double beginLat;
    private Double endLon;
    private Double endLat;
    private LocalDateTime estBeginTime;
    private LocalDateTime estEndTime;
    private LocalDateTime createTime;
    private Integer status;
}
