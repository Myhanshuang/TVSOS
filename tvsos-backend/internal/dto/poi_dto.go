package dto

// 统一设置用来接受请求参数的结构体

// ListPoisReq 用于接受 ListPois 的请求参数
type ListPoisReq struct {
	Name   string `form:"name" json:"name"`
	Tybe   int    `form:"tybe" json:"tybe"`
	Status int    `form:"status" json:"status"`
}
