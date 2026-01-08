package dto;
import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class ModbusVehicleDTO {
    private String license; // 车牌号
    private double lon;     // 经度
    private double lat;     // 纬度
    private double speed;   // 速度
    private long timestamp; // Unix 时间戳 (秒)
}
