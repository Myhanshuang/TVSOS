package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class VehicleCategory {
    private Long Id;
    private String brand;
    private Double length;
    private Double width;
    private Double height;
    private Double capacity;
    private Integer tybe;
    private Integer scope;
}
