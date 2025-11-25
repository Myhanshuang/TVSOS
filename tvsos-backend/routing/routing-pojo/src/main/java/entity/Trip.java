package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Trip {
    private Long id;
    private Long vehicleId;
    private Integer status;
    private LocalDateTime beginTime;
    private LocalDateTime endTime;
    private LocalDateTime createTime;
    private Double beginLat;
    private Double endLat;
    private Double beginLon;
    private Double endLon;
}
