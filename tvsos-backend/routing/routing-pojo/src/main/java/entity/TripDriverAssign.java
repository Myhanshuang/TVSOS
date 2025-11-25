package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class TripDriverAssign {
    private Long id;
    private Long tripId;
    private Long driverId;
    private Integer role;
}
