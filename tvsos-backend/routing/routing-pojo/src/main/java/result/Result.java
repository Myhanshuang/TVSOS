package result;

import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.io.Serializable;

@Data
@AllArgsConstructor
@NoArgsConstructor
public class Result<T> implements Serializable {
    private Integer code;   //1成功 0失败
    private String message;
    private T data;

    public static <T> Result<T> success(){
        Result<T> result = new Result<>();
        result.setCode(1);
        result.setMessage("success");
        return result;
    }

    public static <T> Result<T> success(T data){
        Result<T> result = new Result<>();
        result.setCode(1);
        result.setMessage("success");
        result.setData(data);
        return result;
    }

    public static <T> Result<T> error(String message){
        Result<T> result = new Result<>();
        result.setCode(0);
        result.setMessage(message);
        return result;
    }
}
