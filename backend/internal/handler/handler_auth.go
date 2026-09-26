package handler

import (
	"time"

	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	"github.com/blueship581/cybuildprice/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	secret   string
	validate *validator.Validate
}

func NewAuthHandler(secret string, v *validator.Validate) *AuthHandler {
	return &AuthHandler{secret, v}
}

// DemoToken 为演示环境签发短期角色令牌，让商家侧操作（接单/无法供货）可以走通 RBAC。
func (h *AuthHandler) DemoToken(c *gin.Context) {
	var req dto.DemoTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.Error(err)
		return
	}
	subject := constants.DemoUserID
	if req.Role != constants.RoleUser {
		subject = "demo-" + req.Role
	}
	token, err := middleware.NewDemoToken(h.secret, subject, req.Role)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, gin.H{"token": token, "role": req.Role, "expires_at": time.Now().Add(constants.DemoTokenLifetime)})
}
