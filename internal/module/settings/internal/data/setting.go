package data

import (
	"context"

	v1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/biz"
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/data/ent"
	"github.com/nam3886/kratos-modular/internal/platform/mq"

	"github.com/google/uuid"
)

type settingRepo struct {
	data *Data
}

// NewSettingRepo returns the ent-backed biz.SettingRepo.
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
	return &biz.Setting{Currency: row.ID, MaxAmount: row.MaxAmount}, nil
}

// Save stores the setting and its SettingChanged event in one transaction.
func (r *settingRepo) Save(ctx context.Context, s *biz.Setting) (*biz.Setting, error) {
	err := r.data.inTx(ctx, func(tx *ent.Tx) error {
		_, err := tx.Setting.Get(ctx, s.Currency)
		switch {
		case ent.IsNotFound(err):
			err = tx.Setting.Create().SetID(s.Currency).SetMaxAmount(s.MaxAmount).Exec(ctx)
		case err == nil:
			err = tx.Setting.UpdateOneID(s.Currency).SetMaxAmount(s.MaxAmount).Exec(ctx)
		}
		if err != nil {
			return err
		}
		return mq.Publish(ctx, tx, uuid.NewString(), &v1.SettingChanged{Currency: s.Currency, MaxAmount: s.MaxAmount})
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}
