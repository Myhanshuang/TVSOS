package vo;

import com.alibaba.fastjson2.JSONObject;
import io.swagger.v3.oas.annotations.media.Schema;
import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.LocalDateTime;
import java.util.List;
import com.alibaba.fastjson2.JSONArray;

@Data
@AllArgsConstructor
@NoArgsConstructor
@Schema(description = "VehicleVO")
public class VehicleVO {
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
    
    @Schema(description = "车辆角度 (0-360)")
    private Double angle;

    // 路线距离 单位km
    @Schema(description = "路线距离 单位km")
    private Double distance;
    // 路线用时 单位h
    @Schema(description = "路线用时 单位h")
    private Double duration;
    // 点串 路径点 连点成线
    @Schema(description = "点串 路径点 连点成线")
    private List<Double[]> polyline;
    // 文字阶段性导航
//    private JSONArray steps;
    // 完整json数据
//    private JSONObject raw;
}
