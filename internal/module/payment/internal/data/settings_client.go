package data

import (
	"context"

	settingsv1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"

	"github.com/go-kratos/kratos/v3/errors"
)

type limitReader struct {
	settings settingsv1.SettingsServiceServer
}

// NewLimitReader adapts the settings module's API to biz.LimitReader. The
// call stays in process.
func NewLimitReader(settings settingsv1.SettingsServiceServer) biz.LimitReader {
	return &limitReader{settings: settings}
}

func (r *limitReader) MaxAmount(ctx context.Context, currency string) (int64, error) {
	s, err := r.settings.GetSetting(ctx, &settingsv1.GetSettingRequest{Currency: currency})
	if errors.IsNotFound(err) {
		return 0, biz.ErrPaymentCurrencyNotSupported
	}
	if err != nil {
		return 0, err
	}
	return s.GetMaxAmount(), nil
}
