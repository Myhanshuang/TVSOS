package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Route {
    private Long id;
    private Long beginPoiId;
    private Long endPoiId;
    private Double distance;
    private Integer tybe;
}
