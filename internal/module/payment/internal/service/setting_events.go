package service

import (
	"context"

	settingsv1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"

	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/proto"
)

// SettingEvents handles events published by the settings module.
type SettingEvents struct {
	uc *biz.PaymentUsecase
}

// NewSettingEvents new the settings event handlers.
func NewSettingEvents(uc *biz.PaymentUsecase) *SettingEvents {
	return &SettingEvents{uc: uc}
}

// OnSettingChanged handles settings.v1.SettingChanged.
func (h *SettingEvents) OnSettingChanged(ctx context.Context, msg *message.Message) error {
	var event settingsv1.SettingChanged
	if err := proto.Unmarshal(msg.Payload, &event); err != nil {
		return err
	}
	return h.uc.LimitChanged(ctx, msg.UUID, event.GetCurrency(), event.GetMaxAmount())
}
