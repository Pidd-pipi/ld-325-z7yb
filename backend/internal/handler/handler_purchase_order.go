package handler

import (
	"github.com/blueship581/cybuildprice/backend/internal/constants"
	"github.com/blueship581/cybuildprice/backend/internal/dto"
	apperrors "github.com/blueship581/cybuildprice/backend/internal/errors"
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
	data, err := h.service.Create(c.GetString(constants.UserIDContextKey), req)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, data)
}
func (h *PurchaseOrderHandler) List(c *gin.Context) {
	status := c.Query("status")
	if status != "" && !constants.ValidOrderStatus(status) {
		c.Error(apperrors.ErrInvalidInput)
		return
	}
	data, err := h.service.List(c.GetString(constants.UserIDContextKey), status)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, data)
}
func (h *PurchaseOrderHandler) Accept(c *gin.Context) {
	id, ok := orderID(c)
	if !ok {
		return
	}
	data, err := h.service.Accept(id)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, data)
}
func (h *PurchaseOrderHandler) Reject(c *gin.Context) {
	id, ok := orderID(c)
	if !ok {
		return
	}
	var req dto.RejectPurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.Error(err)
		return
	}
	data, err := h.service.Reject(id, req.Reason)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, data)
}
func (h *PurchaseOrderHandler) Cancel(c *gin.Context) {
	id, ok := orderID(c)
	if !ok {
		return
	}
	data, err := h.service.Cancel(c.GetString(constants.UserIDContextKey), id)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, data)
}
func (h *PurchaseOrderHandler) Receive(c *gin.Context) {
	id, ok := orderID(c)
	if !ok {
		return
	}
	var req dto.ReceiveDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.Error(err)
		return
	}
	data, err := h.service.Receive(c.GetString(constants.UserIDContextKey), id, req)
	if err != nil {
		c.Error(err)
		return
	}
	success(c, data)
}
func orderID(c *gin.Context) (uint, bool) {
	var path struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(err)
		return 0, false
	}
	return path.ID, true
}
