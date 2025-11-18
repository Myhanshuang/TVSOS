package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class ShipmentCargo {
    private Long id;
    private Long shipmentId;
    private Long cargoId;
    private Integer quantity;
    private Double weight; // kg
}
