package entity;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDate;
import java.time.LocalDateTime;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class TransportOrder {
    private Integer id;
    private String orderNumber;
    private Long beginPoiId;
    private Long endPoiId;
    private LocalDate estBeginTime;
    private LocalDate estEndTime;
    private LocalDate actBeginTime;
    private LocalDate actEndTime;
    private LocalDateTime createTime;
    private Integer status;
}
