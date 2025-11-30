package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class TripSegment {
    private Long id;
    private Long tripId;
    private Double beginLon;
    private Double beginLat;
    private Double endLon;
    private Double endLat;
    private Double distance;
    private Integer sequence;
    private Integer status;
    private Double duration;
}
