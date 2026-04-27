package result

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Result 定义了给前端的统一返回对象类型
type Result struct {
	Code    int         `json:"code"` // 1表示业务成功 2表示业务失败
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// gin 框架中给前端返回对象不是通过 return 而是写入 context 对象

// Success 业务成功时返回
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Result{
		Code:    1,
		Message: "success",
		Data:    data,
	})
}

// Fail 业务失败时返回
func Fail(c *gin.Context, msg string, err error) {
	c.JSON(http.StatusOK, Result{
		Code:    2,
		Message: msg,
		Data:    err.Error(),
	})
}
