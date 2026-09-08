package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kiritosuki/mover/internal/model"
	"github.com/kiritosuki/mover/internal/repository"
	"github.com/kiritosuki/mover/internal/result"
	"golang.org/x/crypto/bcrypt"
)

// ---------- 简单演示级 token 管理（内存态，进程重启即失效，无刷新无角色） ----------

type tokenEntry struct {
	UserID    uint
	ExpiresAt time.Time
}

var (
	tokenStore = sync.Map{} // token -> tokenEntry
	tokenTTL   = 24 * time.Hour
)

// issueToken 为用户签发简易 token（crypto/rand 32字节十六进制）
func issueToken(userID uint) string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	token := hex.EncodeToString(buf)
	tokenStore.Store(token, tokenEntry{UserID: userID, ExpiresAt: time.Now().Add(tokenTTL)})
	return token
}

// ResolveToken 校验 token 并返回用户 id（0=无效）。middleware 与 WS 握手共用。
func ResolveToken(token string) uint {
	value, ok := tokenStore.Load(token)
	if !ok {
		return 0
	}
	entry := value.(tokenEntry)
	if time.Now().After(entry.ExpiresAt) {
		tokenStore.Delete(token)
		return 0
	}
	return entry.UserID
}

// revokeToken 注销 token
func revokeToken(token string) {
	tokenStore.Delete(token)
}

// ---------- 请求/响应 ----------

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"displayName"`
}

// Login 登录：校验用户名密码，返回简易 token 与用户信息
// @Summary 用户登录
// @Tags auth
// @Router /auth/login [post]
func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, "用户名与密码必填", err)
		return
	}
	user, err := repository.GetUserByUsername(req.Username)
	if err != nil {
		result.Fail(c, "用户名或密码错误", errors.New("invalid credential"))
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		result.Fail(c, "用户名或密码错误", errors.New("invalid credential"))
		return
	}
	result.Success(c, gin.H{
		"token":       issueToken(user.Id),
		"userId":      user.Id,
		"username":    user.Username,
		"displayName": user.DisplayName,
	})
}

// Register 注册（简单演示级：仅创建账号，无邮箱验证等流程）
// @Summary 用户注册
// @Tags auth
// @Router /auth/register [post]
func Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, "用户名与密码必填", err)
		return
	}
	if _, err := repository.GetUserByUsername(req.Username); err == nil {
		result.Fail(c, "用户名已存在", errors.New("duplicate username"))
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		result.Fail(c, "密码加密失败", err)
		return
	}
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}
	user := &model.User{Username: req.Username, PasswordHash: string(hash), DisplayName: displayName}
	if err := repository.CreateUser(user); err != nil {
		result.Fail(c, "注册失败", err)
		return
	}
	result.Success(c, gin.H{"userId": user.Id, "username": user.Username, "displayName": user.DisplayName})
}

// Profile 当前登录用户信息
// @Summary 当前用户
// @Tags auth
// @Router /auth/profile [get]
func Profile(c *gin.Context) {
	userID := c.GetUint("userId")
	user, err := repository.GetUser(userID)
	if err != nil {
		result.Fail(c, "用户不存在", err)
		return
	}
	result.Success(c, gin.H{"userId": user.Id, "username": user.Username, "displayName": user.DisplayName})
}

// Logout 注销（删除服务端 token）
// @Summary 退出登录
// @Tags auth
// @Router /auth/logout [post]
func Logout(c *gin.Context) {
	if token := c.GetString("authToken"); token != "" {
		revokeToken(token)
	}
	result.Success(c, gin.H{"loggedOut": true})
}
