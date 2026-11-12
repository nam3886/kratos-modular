package biz

import (
	"context"
	"strings"
	"uuid"

	v1 "github.com/nam3886/kratos-modular/api/payment/v1"

	"github.com/go-kratos/kratos/v3/errors"
)

var (
	ErrPaymentNotFound             = errors.NotFound(v1.ErrorReason_PAYMENT_NOT_FOUND.String(), "payment not found")
	ErrPaymentAmountInvalid        = errors.BadRequest(v1.ErrorReason_PAYMENT_AMOUNT_INVALID.String(), "amount must be positive")
	ErrPaymentCurrencyInvalid      = errors.BadRequest(v1.ErrorReason_PAYMENT_CURRENCY_INVALID.String(), "currency must be a 3-letter code")
	ErrPaymentIDInvalid            = errors.BadRequest(v1.ErrorReason_PAYMENT_ID_INVALID.String(), "payment id must be a UUID")
	ErrPaymentCurrencyNotSupported = errors.BadRequest(v1.ErrorReason_PAYMENT_CURRENCY_NOT_SUPPORTED.String(), "currency is not supported")
	ErrPaymentLimitExceeded        = errors.BadRequest(v1.ErrorReason_PAYMENT_LIMIT_EXCEEDED.String(), "amount exceeds the currency limit")
)

type Payment struct {
	ID       uuid.UUID
	Amount   int64
	Currency string
	Status   string
}

// PaymentRepo stores payments. Create also publishes the PaymentCreated event.
type PaymentRepo interface {
	Create(ctx context.Context, p *Payment) (*Payment, error)
	Find(ctx context.Context, id uuid.UUID) (*Payment, error)
}

// LimitReader is what this module needs from the settings module.
type LimitReader interface {
	MaxAmount(ctx context.Context, currency string) (int64, error)
}

// LimitChangeRepo records limit changes this module was told about.
type LimitChangeRepo interface {
	Record(ctx context.Context, messageID, currency string, maxAmount int64) error
}

type PaymentUsecase struct {
	repo    PaymentRepo
	limits  LimitReader
	changes LimitChangeRepo
}

func NewPaymentUsecase(repo PaymentRepo, limits LimitReader, changes LimitChangeRepo) *PaymentUsecase {
	return &PaymentUsecase{repo: repo, limits: limits, changes: changes}
}

func (uc *PaymentUsecase) Create(ctx context.Context, amount int64, currency string) (*Payment, error) {
	currency = strings.ToUpper(currency)
	if amount <= 0 {
		return nil, ErrPaymentAmountInvalid
	}
	if len(currency) != 3 {
		return nil, ErrPaymentCurrencyInvalid
	}
	limit, err := uc.limits.MaxAmount(ctx, currency)
	if err != nil {
		return nil, err
	}
	if amount > limit {
		return nil, ErrPaymentLimitExceeded
	}
	return uc.repo.Create(ctx, &Payment{ID: uuid.NewV7(), Amount: amount, Currency: currency})
}

func (uc *PaymentUsecase) Get(ctx context.Context, id uuid.UUID) (*Payment, error) {
	return uc.repo.Find(ctx, id)
}

func (uc *PaymentUsecase) LimitChanged(ctx context.Context, messageID, currency string, maxAmount int64) error {
	return uc.changes.Record(ctx, messageID, currency, maxAmount)
}
