package handler

import (
	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	"github.com/blueship581/cybuildprice/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type PurchaseOrderHandler struct {
	service  *service.PurchaseOrderService
	validate *validator.Validate
}

func NewPurchaseOrderHandler(s *service.PurchaseOrderService, v *validator.Validate) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{s, v}
}
func (h *PurchaseOrderHandler) Create(c *gin.Context) {
	var req dto.CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.Error(err)
		return
	}
	view, err := h.service.Create(c.GetString(constants.UserIDContextKey), req)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, view)
}
func (h *PurchaseOrderHandler) List(c *gin.Context) {
	views, err := h.service.List(c.GetString(constants.UserIDContextKey))
	if err != nil {
		c.Error(err)
		return
	}
	success(c, views)
}
func (h *PurchaseOrderHandler) Accept(c *gin.Context) {
	var path struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(err)
		return
	}
	view, err := h.service.Accept(path.ID)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, view)
}
func (h *PurchaseOrderHandler) Reject(c *gin.Context) {
	var path struct {
		ID uint `uri:"id" binding:"required"`
	}
	var req dto.RejectPurchaseOrderRequest
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(err)
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.Error(err)
		return
	}
	view, err := h.service.Reject(path.ID, req.Reason)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, view)
}
func (h *PurchaseOrderHandler) Cancel(c *gin.Context) {
	var path struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(err)
		return
	}
	view, err := h.service.Cancel(path.ID, c.GetString(constants.UserIDContextKey))
	if err != nil {
		c.Error(err)
		return
	}
	success(c, view)
}
func (h *PurchaseOrderHandler) RegisterArrival(c *gin.Context) {
	var path struct {
		ID uint `uri:"id" binding:"required"`
	}
	var req dto.RegisterArrivalRequest
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(err)
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.Error(err)
		return
	}
	view, err := h.service.RegisterArrival(path.ID, c.GetString(constants.UserIDContextKey), req)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, view)
}
