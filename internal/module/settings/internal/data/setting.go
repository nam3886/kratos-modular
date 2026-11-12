package data

import (
	"context"
	"uuid"

	v1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/biz"
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/data/ent"
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/data/ent/setting"
	"github.com/nam3886/kratos-modular/internal/platform/mq"
)

type settingRepo struct {
	data *Data
}

func NewSettingRepo(data *Data) biz.SettingRepo {
	return &settingRepo{data: data}
}

func (r *settingRepo) Find(ctx context.Context, currency string) (*biz.Setting, error) {
	row, err := r.data.db.Setting.Get(ctx, currency)
	if ent.IsNotFound(err) {
		return nil, biz.ErrSettingNotFound
	}
	if err != nil {
		return nil, err
	}
	return toBiz(row), nil
}

// Save stores the setting and its SettingChanged event in one transaction.
func (r *settingRepo) Save(ctx context.Context, s *biz.Setting) (*biz.Setting, error) {
	err := r.data.inTx(ctx, func(tx *ent.Tx) error {
		err := tx.Setting.Create().SetID(s.Currency).SetMaxAmount(s.MaxAmount).
			OnConflictColumns(setting.FieldID).UpdateMaxAmount().Exec(ctx)
		if err != nil {
			return err
		}
		return mq.Publish(ctx, tx, uuid.NewV7().String(), &v1.SettingChanged{Currency: s.Currency, MaxAmount: s.MaxAmount})
	})
	if err != nil {
		return nil, err
	}
	return &biz.Setting{Currency: s.Currency, MaxAmount: s.MaxAmount}, nil
}

func toBiz(row *ent.Setting) *biz.Setting {
	return &biz.Setting{Currency: row.ID, MaxAmount: row.MaxAmount}
}
