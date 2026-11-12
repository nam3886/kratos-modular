package mq

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// inject stores the caller's trace context in the message, so the consumer
// continues the trace of the request that published the event.
func inject(ctx context.Context, msg *message.Message) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(msg.Metadata))
}

// trace runs a handler in a span that is a child of the publisher's span.
func trace(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		ctx := otel.GetTextMapPropagator().Extract(msg.Context(), propagation.MapCarrier(msg.Metadata))
		ctx, span := otel.Tracer("mq").Start(ctx, message.HandlerNameFromCtx(ctx), oteltrace.WithSpanKind(oteltrace.SpanKindConsumer))
		defer span.End()
		msg.SetContext(ctx)
		out, err := h(msg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return out, err
	}
}
