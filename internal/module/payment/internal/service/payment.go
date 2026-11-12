package service

import (
	"context"
	"uuid"

	v1 "github.com/nam3886/kratos-modular/api/payment/v1"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"
)

// PaymentService implements api/payment/v1.
type PaymentService struct {
	v1.UnimplementedPaymentServiceServer

	uc *biz.PaymentUsecase
}

// NewPaymentService new a payment service.
func NewPaymentService(uc *biz.PaymentUsecase) *PaymentService {
	return &PaymentService{uc: uc}
}

func (s *PaymentService) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (*v1.Payment, error) {
	p, err := s.uc.Create(ctx, req.GetPayment().GetAmount(), req.GetPayment().GetCurrency())
	if err != nil {
		return nil, err
	}
	return reply(p), nil
}

func (s *PaymentService) GetPayment(ctx context.Context, req *v1.GetPaymentRequest) (*v1.Payment, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, biz.ErrPaymentInvalidArgument
	}
	p, err := s.uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return reply(p), nil
}

func reply(p *biz.Payment) *v1.Payment {
	return &v1.Payment{Id: p.ID.String(), Amount: p.Amount, Currency: p.Currency, Status: p.Status}
}
