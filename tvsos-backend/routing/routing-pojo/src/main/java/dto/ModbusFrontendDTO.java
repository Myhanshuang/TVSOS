package dto;

import com.fasterxml.jackson.annotation.JsonFormat;
import lombok.Data;
import java.time.LocalDateTime;
import lombok.NoArgsConstructor;
import lombok.AllArgsConstructor;
/**
 * 车辆实时状态 DTO (内存中存储并返回给前端)
 */
@Data
@AllArgsConstructor
@NoArgsConstructor
public class ModbusFrontendDTO {
    private String licensePlate;
    private double latitude;
    private double longitude;
    private double speed;
    @JsonFormat(pattern = "yyyy-MM-dd HH:mm:ss")
    private LocalDateTime lastUpdateTime; // 最后更新时间，格式更友好
}
