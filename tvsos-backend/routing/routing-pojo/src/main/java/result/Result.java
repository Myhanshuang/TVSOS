package result;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.io.Serializable;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Result implements Serializable {
    private Integer code;   //1成功 0失败
    private String message;
    private Object data;

    public static Result success(){
        Result result = new Result();
        result.setCode(1);
        result.setMessage("success");
        return result;
    }

    public static Result success(Object data){
        Result result = new Result();
        result.setCode(1);
        result.setMessage("success");
        result.setData(data);
        return result;
    }

    public static Result error(String message){
        Result result = new Result();
        result.setCode(0);
        result.setMessage(message);
        return result;
    }
}
