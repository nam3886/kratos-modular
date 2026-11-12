package biz

import (
	"context"
	"strings"

	v1 "github.com/nam3886/kratos-modular/api/settings/v1"

	"github.com/go-kratos/kratos/v3/errors"
)

var (
	// ErrSettingNotFound is returned when a currency has no setting.
	ErrSettingNotFound = errors.NotFound(v1.ErrorReason_SETTING_NOT_FOUND.String(), "setting not found")
	// ErrSettingInvalidArgument is returned when a setting is invalid.
	ErrSettingInvalidArgument = errors.BadRequest(v1.ErrorReason_SETTING_INVALID_ARGUMENT.String(), "invalid setting")
)

// Setting is the payment limit of one currency.
type Setting struct {
	Currency  string
	MaxAmount int64
}

// SettingRepo stores settings. Save also publishes the change.
type SettingRepo interface {
	Find(ctx context.Context, currency string) (*Setting, error)
	Save(ctx context.Context, s *Setting) (*Setting, error)
}

// SettingUsecase holds the setting rules.
type SettingUsecase struct {
	repo SettingRepo
}

// NewSettingUsecase new a setting usecase.
func NewSettingUsecase(repo SettingRepo) *SettingUsecase {
	return &SettingUsecase{repo: repo}
}

// Get returns the setting of a currency.
func (uc *SettingUsecase) Get(ctx context.Context, currency string) (*Setting, error) {
	return uc.repo.Find(ctx, strings.ToUpper(currency))
}

// Update creates or replaces the setting of a currency.
func (uc *SettingUsecase) Update(ctx context.Context, s *Setting) (*Setting, error) {
	s.Currency = strings.ToUpper(s.Currency)
	if len(s.Currency) != 3 || s.MaxAmount <= 0 {
		return nil, ErrSettingInvalidArgument
	}
	return uc.repo.Save(ctx, s)
}
