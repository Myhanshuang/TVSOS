package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class TripTaskAssign {
    private Long id;
    private Long tripId;
    private Long taskId;
    private Integer sequence;
}
