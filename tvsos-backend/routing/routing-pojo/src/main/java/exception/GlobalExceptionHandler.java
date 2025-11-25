package exception;

import lombok.extern.slf4j.Slf4j;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import result.Result;

@RestControllerAdvice
@Slf4j
public class GlobalExceptionHandler {
    /** 业务异常：明确告诉前端 */
    @ExceptionHandler(ServiceException.class)
    public Result handleServiceException(ServiceException e) {
        log.warn("业务异常：{}", e.getMessage());
        return Result.error(e.getMessage());
    }

    /** 运行时异常：统一兜底 */
    @ExceptionHandler(RuntimeException.class)
    public Result handleRuntimeException(RuntimeException e) {
        log.error("系统异常：", e);
        return Result.error(e.getMessage());
    }
}
