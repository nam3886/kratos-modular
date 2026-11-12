package biz

import (
	"context"
	"strings"

	v1 "github.com/nam3886/kratos-modular/api/settings/v1"

	"github.com/go-kratos/kratos/v3/errors"
)

var (
	ErrSettingNotFound         = errors.NotFound(v1.ErrorReason_SETTING_NOT_FOUND.String(), "setting not found")
	ErrSettingCurrencyInvalid  = errors.BadRequest(v1.ErrorReason_SETTING_CURRENCY_INVALID.String(), "currency must be a 3-letter code")
	ErrSettingMaxAmountInvalid = errors.BadRequest(v1.ErrorReason_SETTING_MAX_AMOUNT_INVALID.String(), "max amount must be positive")
)

type Setting struct {
	Currency  string
	MaxAmount int64
}

// SettingRepo stores settings. Save also publishes the change.
type SettingRepo interface {
	Find(ctx context.Context, currency string) (*Setting, error)
	Save(ctx context.Context, s *Setting) (*Setting, error)
}

type SettingUsecase struct {
	repo SettingRepo
}

func NewSettingUsecase(repo SettingRepo) *SettingUsecase {
	return &SettingUsecase{repo: repo}
}

func (uc *SettingUsecase) Get(ctx context.Context, currency string) (*Setting, error) {
	return uc.repo.Find(ctx, strings.ToUpper(currency))
}

func (uc *SettingUsecase) Update(ctx context.Context, s *Setting) (*Setting, error) {
	s.Currency = strings.ToUpper(s.Currency)
	if len(s.Currency) != 3 {
		return nil, ErrSettingCurrencyInvalid
	}
	if s.MaxAmount <= 0 {
		return nil, ErrSettingMaxAmountInvalid
	}
	return uc.repo.Save(ctx, s)
}
