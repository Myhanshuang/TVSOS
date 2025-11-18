package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Cargo {
    private Long id;
    private String name;
    private Integer pack;   // 1箱 2捆 3袋 4桶
    private Integer level;  // 1普通 2冷链 3危险品
    private Double weight;  // 单件重量（新增）
}