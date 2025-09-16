package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Poi {
    private Long id;
    private String name;
    private Double lon;
    private Double lat;
    private Integer type;
    private Integer status;
}
