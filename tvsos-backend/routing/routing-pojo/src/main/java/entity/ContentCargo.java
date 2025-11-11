package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;
@Data
@AllArgsConstructor
@NoArgsConstructor
//这个类装某一个任务具体的货物和数量信息
public class ContentCargo {
    private Long id;
    private Long transportOrderId;
    private Long cargoId;
    private Integer quantity;
    private LocalDateTime createTime;
}
