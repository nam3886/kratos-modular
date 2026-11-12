package service

import (
	"context"
	"uuid"

	v1 "github.com/nam3886/kratos-modular/api/payment/v1"
	"github.com/nam3886/kratos-modular/internal/module/payment/internal/biz"

	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
)

type PaymentService struct {
	v1.UnimplementedPaymentServiceServer

	uc *biz.PaymentUsecase
}

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
		return nil, biz.ErrPaymentIDInvalid
	}
	p, err := s.uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return reply(p), nil
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// listFields are the fields ListPayments may filter and order on.
var listFields, _ = filtering.NewDeclarations(
	filtering.DeclareStandardFunctions(),
	filtering.DeclareIdent("amount", filtering.TypeInt),
	filtering.DeclareIdent("currency", filtering.TypeString),
	filtering.DeclareIdent("status", filtering.TypeString),
)

func (s *PaymentService) ListPayments(ctx context.Context, req *v1.ListPaymentsRequest) (*v1.ListPaymentsResponse, error) {
	filter, err := filtering.ParseFilter(req, listFields)
	if err != nil {
		return nil, biz.ErrPaymentFilterInvalid
	}
	orderBy, err := ordering.ParseOrderBy(req)
	if err == nil {
		err = orderBy.ValidateForPaths("amount", "currency", "status")
	}
	if err != nil {
		return nil, biz.ErrPaymentOrderByInvalid
	}
	page, size := max(int(req.GetPage()), 1), int(req.GetPageSize())
	if size <= 0 {
		size = defaultPageSize
	}
	size = min(size, maxPageSize)
	payments, total, err := s.uc.List(ctx, biz.ListQuery{Filter: filter, OrderBy: orderBy, Offset: (page - 1) * size, Limit: size})
	if err != nil {
		return nil, err
	}
	res := &v1.ListPaymentsResponse{TotalSize: int32(total), Payments: make([]*v1.Payment, 0, len(payments))}
	for _, p := range payments {
		res.Payments = append(res.Payments, reply(p))
	}
	return res, nil
}

func reply(p *biz.Payment) *v1.Payment {
	return &v1.Payment{Id: p.ID.String(), Amount: p.Amount, Currency: p.Currency, Status: p.Status}
}
