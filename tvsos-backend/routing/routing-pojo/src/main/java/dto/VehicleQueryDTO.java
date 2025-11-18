package dto;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class VehicleQueryDTO {
    private Integer status;
    private String license;
    private Long categoryId;
}
