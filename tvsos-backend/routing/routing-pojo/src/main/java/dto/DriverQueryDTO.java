package dto;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class DriverQueryDTO {
    private String name;
    private Integer level;
    private String phone;
    private Integer status;
}
