package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Assign {
    private Long id;
    private Long vehicleId;
    private Long driverId;
    private Long transportId;
}
