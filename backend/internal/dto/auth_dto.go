package dto

type DemoTokenRequest struct {
	Role string `json:"role" validate:"required,oneof=admin supplier user"`
}
