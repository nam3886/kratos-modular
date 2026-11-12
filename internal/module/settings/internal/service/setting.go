package service

import (
	"context"

	v1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/biz"
)

type SettingsService struct {
	v1.UnimplementedSettingsServiceServer

	uc *biz.SettingUsecase
}

func NewSettingsService(uc *biz.SettingUsecase) *SettingsService {
	return &SettingsService{uc: uc}
}

func (s *SettingsService) GetSetting(ctx context.Context, req *v1.GetSettingRequest) (*v1.Setting, error) {
	got, err := s.uc.Get(ctx, req.GetCurrency())
	if err != nil {
		return nil, err
	}
	return reply(got), nil
}

func (s *SettingsService) UpdateSetting(ctx context.Context, req *v1.UpdateSettingRequest) (*v1.Setting, error) {
	got, err := s.uc.Update(ctx, &biz.Setting{Currency: req.GetSetting().GetCurrency(), MaxAmount: req.GetSetting().GetMaxAmount()})
	if err != nil {
		return nil, err
	}
	return reply(got), nil
}

func reply(s *biz.Setting) *v1.Setting {
	return &v1.Setting{Currency: s.Currency, MaxAmount: s.MaxAmount}
}
