package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class OrderRoute {
    private Long id;
    private Long transportOrderId;
    private Long routeId;
    private Integer sequence;
    private Long status;
}
