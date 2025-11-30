package entity;


import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Vehicle {
    private Long id;
    private String license;
    private Integer status;
    private Double lon;
    private Double lat;
    private Double speed;
    private LocalDateTime createTime;
    private LocalDateTime updateTime;
    private Long categoryId;
    private Double cargoSize;
}
