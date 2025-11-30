package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Task {
    private Long id;
    private Long shipmentId;
    private Long cargoId;
    private Integer quantity;
    private Double beginLon;
    private Double beginLat;
    private Double endLon;
    private Double endLat;
    private Integer status;
    private Double weight;
    private LocalDateTime createTime;
}
