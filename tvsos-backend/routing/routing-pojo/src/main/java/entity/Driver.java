package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Driver {
    private Long id;
    private String name;
    private Integer level;
    private Integer status;
    private String phone;
}
