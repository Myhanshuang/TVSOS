# swagger注释样例

模版：

```go
// ListVehicles godoc
// @Summary 筛选/获取车辆列表
// @Description 根据条件筛选/获取车辆列表
// @Tags Vehicle
// @Accept json
// @Produce json
// @Param license query string false "车牌号"
// @Param status query int false "状态"
// @Success 200 {object} common.Result{data=[]model.Vehicle}
// @Failure 400 {object} common.Result
// @Router /vehicles [get]
```

注释需写在api层

- `ListVehicles godoc`：必须，方法名 + godoc

- `@Summary`：简述接口

- `@Descripption`：详细描述接口功能

- `@Tags`：指定标签，一般和router对应即可，swagger会根据tags自动分类

- `@Accept`：接收格式，一般写json

- `@Produce`：响应格式，一般写json

- `@Param`：参数说明，格式如下

  `@Param <name> <from> <type> <required> "<description>"`

  - `name`：参数名称
  - `from`：参数来源，可取以下值：
    1. `query`：url查询参数，例：`/vehicles?status=1`
    2. `path`：路径参数，例：`/vehicles/1`
    3. `body`：请求体
    4. `header`：请求头
  - `type`：数据类型
  - `required`：true / false，是否必须
  - `<description>`：自定义参数备注 / 描述信息

- `@Success`：成功时返回的数据格式，一般只需要改`Result{data=xxx}`

- `@Failure` ：失败时返回的数据格式，一般不用改

- `@Router`：`url [请求方式]`，和router保持一致即可