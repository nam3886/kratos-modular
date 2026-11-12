# Spec bổ sung: quan sát và API danh sách

Nov 12, 2026 · @Nguyen Phuong Nam

Spec này thêm health check, log request dạng JSON, tracing xuyên outbox, metrics cho database và messaging, và API `ListPayments` phân trang offset vào `kratos-modular`. Mọi đoạn code dưới đây đã chạy thật trên bản spike: build, lint 0 lỗi, toàn bộ test qua với `-race`.

## 1. Phạm vi và điểm xuất phát

Làm tiếp trên nhánh của PR #1 (`https://github.com/nam3886/kratos-modular/pull/1`), commit đầu `8f9af51`. Spec gốc "Spec: kratos-modular boilerplate" vẫn có hiệu lực; tài liệu này chỉ mô tả phần thay đổi. Chỗ nào hai tài liệu khác nhau thì theo tài liệu này.

File `kratos-modular-next.patch` đi kèm là bản diff của spike so với `8f9af51` (21 file, +472/−9). Dùng nó làm bản tham chiếu: áp dụng được bằng `git apply`, nhưng không chứa `go.mod`, `go.sum` và mã sinh tự động. Sau khi áp dụng phải chạy `make generate` và `go mod tidy`.

**Mười lăm việc cần làm**

| # | Nhóm | Việc |
| --- | --- | --- |
| 1 | Health | `GET /healthz` luôn trả 200 |
| 2 | Health | `GET /readyz` ping từng database, không kiểm tra RabbitMQ |
| 3 | Log | Log dạng JSON; mỗi request một dòng có nội dung request |
| 4 | Log | Client ngắt kết nối trả 499 `CLIENT_CLOSED` |
| 5 | Metrics | Pool của từng database |
| 6 | Metrics | Số event đang chờ trong outbox |
| 7 | Metrics | Messaging: publish, nhận, xử lý (tiền tố `mq_`) |
| 8 | Trace | Tracer provider và middleware tracing cho HTTP, gRPC |
| 9 | Trace | Span SQL qua `otelsql`, có lọc |
| 10 | Trace | Trace xuyên outbox → RabbitMQ → consumer |
| 11 | Trace | Xuất OTLP/gRPC qua khoá `trace.endpoint` |
| 12 | API | `ListPayments` với `page`, `page_size`, `filter`, `order_by` |
| 13 | API | Lọc và sắp xếp bằng `aip-go/ents` |
| 14 | API | Lỗi filter hoặc `order_by` trả 400 `PAYMENT_LIST_INVALID` |
| 15 | Tài liệu | Cập nhật README và AGENTS.md |

**Nguyên tắc giữ nguyên từ spec gốc**: chỉ `config.yaml` là nguồn cấu hình; không thêm quy ước, arch test hay package dùng chung mới; boilerplate phải gọn. Bản spike thêm 472 dòng, kể cả test và proto. Nếu bản làm vượt xa con số này thì đang làm thừa.

## 2. Thư viện và phiên bản

Thêm bốn thư viện trực tiếp. Các gói `go.opentelemetry.io/otel`, `otel/sdk`, `otel/trace` đã có sẵn ở dạng gián tiếp và sẽ thành trực tiếp.

| Thư viện | Phiên bản đã chạy | Dùng cho |
| --- | --- | --- |
| `github.com/XSAM/otelsql` | v0.44.0 | Span SQL |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | cùng bản với `otel` đang dùng (v1.47.0) | Xuất trace |
| `github.com/go-kratos/aip-go/ents` | v0.0.0-20260901003907-1ab5471e605b | Dịch filter và order\_by sang Ent |
| `go.einride.tech/aip` | v0.76.0 trở lên | Parse filter và order\_by |

`aip-go/ents` chưa có bản gắn tag; ghim đúng commit `1ab5471`, là commit kratos-layout đang dùng. Các phiên bản còn lại để `go mod tidy` chọn, miễn không thấp hơn bảng trên.

Gói semconv: spike dùng `go.opentelemetry.io/otel/semconv/v1.26.0` với `semconv.ServiceName`, `semconv.ServiceVersion`, `semconv.DBSystemPostgreSQL`, `semconv.DBNamespace`. Được dùng bản semconv mới hơn nếu các hằng tương đương tồn tại.

## 3. Cấu hình

Thêm một khoá duy nhất. Không thêm biến môi trường.

**`internal/conf/conf.proto`**

```protobuf
message Bootstrap {
  Server server = 1;
  Data data = 2;
  Trace trace = 3;
}

message Trace {
  // OTLP/gRPC address of the trace collector, e.g. "localhost:4317".
  // Empty: traces are not exported; logs still carry trace ids.
  string endpoint = 1;
}
```

**`configs/config.yaml`**: thêm khối dưới đây, để trống giá trị.

```yaml
trace:
  endpoint: ""
```

Kết nối tới collector không mã hoá (`WithInsecure`). TLS và lấy mẫu theo tỷ lệ nằm ngoài phạm vi.

## 4. Health check

Hai đường dẫn HTTP thuần, đăng ký trong `NewHTTPServer` ngay sau `/metrics`. Không khai báo trong proto, không xuất hiện trong OpenAPI, không đi qua middleware nên không sinh log, metrics hay trace.

| Đường dẫn | Hành vi |
| --- | --- |
| `GET /healthz` | Luôn trả 200, thân rỗng |
| `GET /readyz` | Ping từng database bằng context của request. Tất cả thành công: 200. Một database lỗi: 503, thân `database <tên> unreachable`, ghi log ERROR `not ready` |

`/readyz` không kiểm tra RabbitMQ. Nhờ outbox, API vẫn chạy đúng khi broker chết; nếu readiness phụ thuộc broker thì broker sập sẽ kéo mọi instance ra khỏi load balancer.

```go
srv.Handle("/metrics", m.handler)
srv.HandleFunc("/healthz", func(w stdhttp.ResponseWriter, _ *stdhttp.Request) { w.WriteHeader(stdhttp.StatusOK) })
srv.HandleFunc("/readyz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	for _, d := range dbs.All() {
		if err := d.Pool.PingContext(r.Context()); err != nil {
			logger.ErrorContext(r.Context(), "not ready", "database", d.Name, "error", err)
			stdhttp.Error(w, "database "+d.Name+" unreachable", stdhttp.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(stdhttp.StatusOK)
})
```

gRPC đã có health service do Kratos tự đăng ký; không làm gì thêm.

## 5. Log

Log ra stdout dạng JSON, mỗi request một dòng, có nội dung request và `trace_id`.

**`cmd/server/main.go`**: đổi `slog.NewTextHandler` thành `slog.NewJSONHandler`. Giữ nguyên `AddSource`, `log.WithExtractor(tracing.TraceAttrs)` và ba trường `service.*`.

**`internal/server/log.go`** (file mới). Không dùng `logging.Server` có sẵn của Kratos vì nó ghi mọi lỗi, kể cả 400, ở mức ERROR.

```go
package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/logging"
	"github.com/go-kratos/kratos/v3/transport"
)

// maxArgsLogged caps the request text written to the log.
const maxArgsLogged = 2000

// requestLog writes one line per request: ERROR for server faults, WARN for
// client faults, INFO otherwise. A request type that carries secrets hides
// them by implementing logging.Redacter.
func requestLog(logger *slog.Logger) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			start := time.Now()
			reply, err := next(ctx, req)
			var kind, op string
			if tr, ok := transport.FromServerContext(ctx); ok {
				kind, op = tr.Kind().String(), tr.Operation()
			}
			code := 200
			level := slog.LevelInfo
			if se := errors.FromError(err); se != nil {
				code = int(se.Code)
				level = slog.LevelWarn
				if code >= 500 {
					level = slog.LevelError
				}
			}
			logger.LogAttrs(ctx, level, "request",
				slog.String("kind", kind),
				slog.String("operation", op),
				slog.String("args", args(req)),
				slog.Int("code", code),
				slog.String("reason", errors.Reason(err)),
				slog.Duration("latency", time.Since(start)),
			)
			return reply, err
		}
	}
}

func args(req any) string {
	var s string
	switch v := req.(type) {
	case logging.Redacter:
		s = v.Redact()
	case fmt.Stringer:
		s = v.String()
	default:
		s = fmt.Sprintf("%+v", req)
	}
	if len(s) > maxArgsLogged {
		s = s[:maxArgsLogged] + "..."
	}
	return s
}
```

Hành vi bắt buộc:

- Mức log: INFO khi thành công, WARN cho mã dưới 500, ERROR từ 500 trở lên.
- `args` là nội dung request, cắt ở 2.000 ký tự. Message nào có trường nhạy cảm thì tự khai báo `Redact() string`; boilerplate không có message nào như vậy nên không cần viết mẫu.
- Dòng log có `trace_id` và `span_id` nhờ `tracing.TraceAttrs`, vì middleware tracing chạy trước (mục 6).

**`internal/server/errors.go`**: trong `hideInternal`, trước đoạn ghi log "internal error", thêm:

```go
if errors.Is(ctx.Err(), context.Canceled) {
	return nil, errors.ClientClosed("CLIENT_CLOSED", "client closed the request")
}
```

Nhờ đó client tự ngắt kết nối cho ra mã 499 ở mức WARN thay vì 500 ở mức ERROR. Lỗi Kratos vẫn được trả nguyên như cũ.

## 6. Tracing

Một request cho ra một trace gồm: span của request, các span SQL của nó, span của mỗi consumer xử lý event mà request phát ra, và các span SQL của consumer. Spike đã thấy đúng cây này với 12 span cho hai request.

### 6.1 Tracer provider (`internal/server/tracing.go`, file mới)

```go
package server

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/nam3886/kratos-modular/internal/conf"
)

// SetupTracing installs the global tracer provider. Every request then has a
// trace id (visible in logs); spans are exported only when an endpoint is set.
func SetupTracing(ctx context.Context, c *conf.Trace, name, version string) (func(), error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(resource.NewSchemaless(semconv.ServiceName(name), semconv.ServiceVersion(version))),
	}
	if c.GetEndpoint() != "" {
		exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(c.GetEndpoint()), otlptracegrpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
	}, nil
}
```

`main.go` gọi hàm này sau khi đọc config và trước `wireApp`, rồi `defer` hàm trả về:

```go
stopTracing, err := server.SetupTracing(context.Background(), bc.Trace, Name, Version)
if err != nil {
	panic(err)
}
defer stopTracing()
```

Dùng provider toàn cục của OpenTelemetry là có chủ ý: middleware Kratos, `otelsql` và `mq` đều đọc từ đó, nên không phải truyền thêm tham số qua Wire.

### 6.2 Middleware

Thứ tự middleware của cả HTTP và gRPC, bắt buộc đúng thứ tự này:

```go
recovery.Recovery(),
tracing.Server(),      // github.com/go-kratos/kratos/contrib/otel/v3/tracing
requestLog(logger),
m.middleware,          // metrics
hideInternal,
```

`tracing.Server()` đứng trước `requestLog` để dòng log có `trace_id`. `requestLog` đứng ngoài `hideInternal` để ghi mã lỗi cuối cùng mà client nhận.

### 6.3 Span SQL (`internal/platform/db/db.go`)

Thay `sql.Open` bằng `otelsql.Open`:

```go
pool, err := otelsql.Open(dc.GetDriver(), dc.GetSource(),
	otelsql.WithAttributes(semconv.DBSystemPostgreSQL, semconv.DBNamespace(name)),
	// Only statements that belong to a request or a message are traced;
	// the outbox poll would otherwise add a trace every 200ms.
	otelsql.WithSpanOptions(otelsql.SpanOptions{
		OmitConnResetSession: true,
		OmitConnectorConnect: true,
		OmitRows:             true,
		SpanFilter: func(ctx context.Context, _ otelsql.Method, _ string, _ []driver.NamedValue) bool {
			return trace.SpanContextFromContext(ctx).IsValid()
		},
	}))
```

`SpanFilter` là bắt buộc. Không có nó, vòng quét outbox sinh khoảng 25 span mỗi giây không thuộc request nào.

### 6.4 Trace xuyên outbox (`internal/platform/mq/trace.go`, file mới)

```go
package mq

import (
	"context"
	"database/sql"
	"time"

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
```

File này còn chứa hàm `Pending` của mục 7.2.

Hai chỗ gọi:

- `outbox.go`, trong `Publish`: gọi `inject(ctx, msg)` ngay sau `msg.SetContext(ctx)`.
- `mq.go`, trong `New`: đặt `trace` làm middleware đầu tiên của router, trước `poison`. Một span vì thế bao trọn mọi lần retry của một message.

Forwarder của Watermill giữ nguyên metadata khi chuyển từ bảng outbox sang RabbitMQ, nên không phải sửa gì ở forwarder. Tên span của consumer có dạng `payment:settings.v1.SettingChanged`.

## 7. Metrics

Mọi metrics vẫn nằm trong một registry riêng và ra ở `GET /metrics`. Registry trở thành một provider của Wire để `mq` cũng đăng ký được vào đó.

| Metrics | Nguồn | Nhãn |
| --- | --- | --- |
| `go_sql_open_connections`, `go_sql_in_use_connections`, `go_sql_idle_connections`, `go_sql_max_open_connections`, `go_sql_wait_count_total`, `go_sql_wait_duration_seconds_total` | `collectors.NewDBStatsCollector` | `db_name` |
| `outbox_pending_messages` | `GaugeFunc` tự viết | `db_name` |
| `mq_publish_time_seconds` | Watermill, trên publisher của forwarder | `handler_name`, `publisher_name`, `success` |
| `mq_subscriber_messages_received_total` | Watermill | `handler_name`, `subscriber_name`, `acked` |
| `mq_handler_execution_time_seconds` | Watermill | `handler_name`, `success` |

Các metrics request (`server_requests_code_total`, `server_requests_seconds`) và runtime giữ nguyên.

### 7.1 Registry (`internal/server/metrics.go`)

Tách việc tạo registry ra khỏi `NewMetrics`:

```go
// NewRegistry holds every metric of the application: runtime, connection
// pools and the outbox backlog here, requests and messaging elsewhere.
func NewRegistry(dbs *db.Manager) *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	for _, d := range dbs.All() {
		registry.MustRegister(
			collectors.NewDBStatsCollector(d.Pool, d.Name),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name:        "outbox_pending_messages",
				Help:        "Events committed but not yet delivered to RabbitMQ.",
				ConstLabels: prometheus.Labels{"db_name": d.Name},
			}, func() float64 { return mq.Pending(d.Pool) }),
		)
	}
	return registry
}

func NewMetrics(registry *prometheus.Registry) (*Metrics, error) { /* phần còn lại như cũ */ }
```

Thêm `NewRegistry` vào `server.ProviderSet`.

### 7.2 Số event đang chờ (`internal/platform/mq/trace.go`)

```go
// Pending counts the events waiting in the outbox of one database.
func Pending(pool *sql.DB) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var n float64
	if err := pool.QueryRowContext(ctx, `SELECT count(*) FROM "watermill_`+outboxTopic+`"`).Scan(&n); err != nil {
		return -1
	}
	return n
}
```

Giá trị −1 nghĩa là không đọc được. Truy vấn chạy mỗi lần Prometheus quét. Được đặt hàm này ở file khác trong package `mq` nếu thấy hợp hơn.

### 7.3 Metrics của Watermill (`internal/platform/mq/mq.go`)

`New` nhận thêm `registry *prometheus.Registry`. Trong `New`, trước `router.AddMiddleware`:

```go
metrics := wmetrics.NewPrometheusMetricsBuilder(registry, "mq", "") // watermill/components/metrics
metrics.AddPrometheusRouterMetrics(router)
fwdPub, err := metrics.DecoratePublisher(pub)
if err != nil {
	return nil, nil, err
}
```

Truyền `fwdPub` thay cho `pub` vào `forwarder.NewForwarder`. Publisher dùng cho poison queue vẫn là `pub`.

## 8. API `ListPayments`

API danh sách mẫu, phân trang theo offset, lọc và sắp xếp theo cú pháp AIP. Toàn bộ nằm trong module `payment`; không tạo package dùng chung.

```
GET /v1/payments?page=2&page_size=10&filter=currency="USD" AND amount>200&order_by=amount desc
→ { "payments": [...], "totalSize": 29 }
```

| Tham số | Quy tắc |
| --- | --- |
| `page` | Bắt đầu từ 1. Không gửi hoặc nhỏ hơn 1 thì coi là 1 |
| `page_size` | Mặc định 20. Lớn hơn 100 thì hạ về 100, không báo lỗi |
| `filter` | Cú pháp AIP-160 trên `amount` (số nguyên), `currency`, `status` (chuỗi) |
| `order_by` | Cú pháp AIP-132 trên `amount`, `currency`, `status` |
| `totalSize` | Số bản ghi khớp filter trên mọi trang |

Thứ tự luôn kết thúc bằng `id` giảm dần. Không gửi `order_by` thì bản ghi mới nhất đứng đầu, vì id là UUIDv7. Trang vượt quá dữ liệu trả danh sách rỗng kèm `totalSize` đúng.

### 8.1 Proto (`api/payment/v1/payment.proto`)

```protobuf
rpc ListPayments (ListPaymentsRequest) returns (ListPaymentsResponse) {
  option (google.api.http) = {get: "/v1/payments"};
}

message ListPaymentsRequest {
  // Page number, starting at 1. Default 1.
  int32 page = 1;
  // Items per page. Default 20, at most 100.
  int32 page_size = 2;
  // AIP-160 filter on amount, currency, status, e.g. `currency="USD" AND amount>100`.
  string filter = 3;
  // AIP-132 order on amount, currency, status, e.g. `amount desc`. Default: newest first.
  string order_by = 4;
}

message ListPaymentsResponse {
  repeated Payment payments = 1;
  // Number of payments matching the filter, across all pages.
  int32 total_size = 2;
}
```

Thêm `PAYMENT_LIST_INVALID = 7;` vào `ErrorReason`.

### 8.2 `biz`

```go
ErrPaymentFilterInvalid  = errors.BadRequest(v1.ErrorReason_PAYMENT_LIST_INVALID.String(), "filter is not valid; fields: amount, currency, status")
ErrPaymentOrderByInvalid = errors.BadRequest(v1.ErrorReason_PAYMENT_LIST_INVALID.String(), "order_by is not valid; fields: amount, currency, status")

type ListQuery struct {
	Filter  filtering.Filter // go.einride.tech/aip/filtering
	OrderBy ordering.OrderBy // go.einride.tech/aip/ordering
	Offset  int
	Limit   int
}

// trong PaymentRepo:
// List returns one page and the number of payments matching the filter.
List(ctx context.Context, q ListQuery) ([]*Payment, int, error)

func (uc *PaymentUsecase) List(ctx context.Context, q ListQuery) ([]*Payment, int, error) {
	return uc.repo.List(ctx, q)
}
```

### 8.3 `service`

```go
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
```

Lỗi parse của thư viện không được trả nguyên văn cho client: nó là lỗi Go thường (sẽ thành 500 qua `hideInternal`) và thông báo cú pháp dài hàng chục dòng.

### 8.4 `data`

```go
// List orders by id last: offset pagination needs a total order, and
// UUIDv7 ids put the newest first when nothing else is asked for.
func (r *paymentRepo) List(ctx context.Context, q biz.ListQuery) ([]*biz.Payment, int, error) {
	query := r.data.db.Payment.Query().Where(ents.ApplyFilter(q.Filter)) // github.com/go-kratos/aip-go/ents
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.
		Order(ents.ApplyOrderBy(q.OrderBy), payment.ByID(sql.OrderDesc())). // entgo.io/ent/dialect/sql
		Offset(q.Offset).
		Limit(q.Limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.Payment, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBiz(row))
	}
	return out, total, nil
}
```

Không cần migration mới: schema không đổi. Sau khi sửa proto chạy `make api` để sinh lại mã và `api/openapi/openapi.yaml`.

## 9. Wiring

Bốn chữ ký hàm thay đổi và một provider mới. `wireApp` giữ nguyên chữ ký `(*conf.Server, *conf.Data, *slog.Logger)`.

| Hàm | Chữ ký mới |
| --- | --- |
| `server.NewRegistry` (mới) | `func(dbs *db.Manager) *prometheus.Registry` |
| `server.NewMetrics` | `func(registry *prometheus.Registry) (*Metrics, error)` |
| `server.NewHTTPServer` | `func(c *conf.Server, logger *slog.Logger, m *Metrics, dbs *db.Manager, modules []Registrar) *http.Server` |
| `server.NewGRPCServer` | `func(c *conf.Server, logger *slog.Logger, m *Metrics, modules []Registrar) *grpc.Server` |
| `mq.New` | `func(c *conf.Data, dbs *db.Manager, log *slog.Logger, registry *prometheus.Registry) (*MQ, func(), error)` |

`cmd/server/wire.go` không phải sửa: Wire tự nối `*prometheus.Registry` theo kiểu. Chạy `make wire` để sinh lại `wire_gen.go`.

Tracing không đi qua Wire (mục 6.1). Test integration gọi thẳng `wireApp` nên chạy không có tracer provider; điều đó chấp nhận được.

## 10. Test

Thêm đúng ba test; mã đầy đủ có trong file patch. Test cũ chỉ sửa chỗ gọi `mq.New` (thêm `prometheus.NewRegistry()`).

| Test | File | Kiểm tra |
| --- | --- | --- |
| `TestHideInternalClientClosed` | `internal/server/errors_test.go` | Context đã huỷ thì `hideInternal` trả mã 499, reason `CLIENT_CLOSED` |
| `TestListFiltersOrdersAndPages` | `internal/module/payment/internal/data/payment_test.go` | PostgreSQL thật: tạo 3 USD và 1 EUR; lọc `currency="USD"`, sắp `amount desc`, offset 2, limit 2 cho ra total 3 và đúng bản ghi 10 USD; không sắp xếp thì bản mới nhất đứng đầu |
| `TestTraceCrossesTheOutbox` | `internal/platform/mq/trace_test.go` | PostgreSQL và RabbitMQ thật: publish một event trong transaction dưới một span; handler nhận được cùng trace id |

Các fake repository trong test của `biz` và `service` nhúng interface `PaymentRepo` nên không phải thêm phương thức `List`.

Không viết test cho `/healthz`, `/readyz`, `requestLog`, metrics và `SetupTracing`: chúng được kiểm bằng tiêu chí nghiệm thu ở mục 13.

## 11. Tài liệu

**`README.md`**

- Danh sách tính năng đầu file: dòng Operations ghi thêm health check, log JSON và tracing.
- Ví dụ curl: thêm một lệnh gọi `GET /v1/payments` có `page_size`, `filter`, `order_by`.
- Bảng Configuration: thêm `trace.endpoint`.
- Mục Operations, thêm các dòng:
  - `/healthz` và `/readyz`, kèm ghi chú `/readyz` không phụ thuộc RabbitMQ.
  - Các metrics mới ở mục 7 và ý nghĩa: `outbox_pending_messages` tăng liên tục nghĩa là forwarder hoặc broker có vấn đề.
  - Log là JSON, mỗi request một dòng có nội dung request cắt ở 2.000 ký tự; message có dữ liệu nhạy cảm phải khai báo `Redact() string`.
  - Lọc log theo `trace_id` để xem toàn bộ diễn biến của một request, kể cả phần consumer xử lý event.
  - Bảng `inbox` không tự dọn: xoá định kỳ các dòng cũ theo `created_at`.
  - Hàng đợi `poison_poison` không có ai đọc: cần theo dõi độ dài và xử lý tay.

**`AGENTS.md`**

- Mục kiến trúc hoặc mục tương ứng: một đoạn ngắn về API danh sách, trỏ tới `ListPayments` làm mẫu (khai báo trường được lọc ở `service`, dịch sang Ent ở `data`, luôn thêm `id` làm khoá sắp xếp cuối).
- Một dòng: event publish qua `mq.Publish` tự mang trace của request; handler dùng `ctx` được truyền vào để log và truy vấn.

Không thêm file tài liệu mới.

## 12. Bẫy đã gặp khi spike

1. **Không lọc span SQL thì trace ngập rác.** Vòng quét outbox mỗi 200ms sinh span không có cha. Một request cho ra 256 span khi chưa lọc, 12 span sau khi lọc (mục 6.3).
2. **`logging.Server` của Kratos ghi lỗi 400 ở mức ERROR.** Vì vậy dùng `requestLog` tự viết (mục 5).
3. **Lỗi parse filter là lỗi Go thường.** Không đổi sang lỗi Kratos thì client nhận 500. Thông báo lỗi cú pháp của thư viện dài hàng chục dòng, không đưa cho client (mục 8.3).
4. **Metrics của Watermill phải gắn trước khi tạo forwarder.** Forwarder có router riêng, nên publisher của nó phải được bọc bằng `DecoratePublisher` trước khi truyền vào (mục 7.3).
5. **Nhãn `publisher_name` của `mq_publish_time_seconds` hiện là `message.disabledPublisher`.** Đó là tên nội bộ của Watermill; không sửa.
6. **`go.mod` của PR có dòng `golang.org/x/tools/go/packages/packagestest v0.1.1-deprecated // indirect`.** Để `go mod tidy` tự quyết định giữ hay bỏ.
7. **Query string nhận cả `page_size` lẫn `pageSize`.** Không phải làm gì thêm; README dùng dạng `page_size`.
8. **`Count` và truy vấn trang là hai câu SQL riêng**, không nằm trong transaction. `totalSize` có thể lệch một vài bản ghi khi có ghi đồng thời; chấp nhận được.

## 13. Tiêu chí nghiệm thu

Các tiêu chí A–I của spec gốc vẫn phải đạt. Thêm:

**J. Build và test**

- [ ] `make generate` không tạo ra thay đổi nào so với mã đã commit.
- [ ] `go build ./...`, `go vet -tags integration ./...`, `make lint` sạch.
- [ ] `make test-integration` qua, kể cả với `-race`, gồm ba test mới ở mục 10.

**K. Health**

- [ ] `GET /healthz` trả 200.
- [ ] `GET /readyz` trả 200 khi database chạy; dừng PostgreSQL thì trả 503 và log có dòng `not ready`.
- [ ] Dừng RabbitMQ: `/readyz` vẫn 200.
- [ ] Hai đường dẫn này không sinh dòng log `request` và không có trong `server_requests_code_total`.

**L. Log**

- [ ] Mọi dòng stdout của server là JSON hợp lệ.
- [ ] Mỗi request API sinh đúng một dòng `"msg":"request"` có `operation`, `args`, `code`, `reason`, `latency`, `trace_id`.
- [ ] Request thành công ở mức INFO; request trả 400 ở mức WARN; lỗi nội bộ ở mức ERROR kèm một dòng `internal error` có lỗi gốc.
- [ ] Client ngắt kết nối giữa chừng: dòng `request` có `code` 499, không có dòng `internal error`.

**M. Tracing**

- [ ] `trace.endpoint` để trống: server chạy bình thường, log vẫn có `trace_id` khác rỗng.
- [ ] Gửi `PUT /v1/settings/eur` kèm header `traceparent: 00-aaaabbbbccccddddeeeeffff00001111-1234567890abcdef-01`: dòng log của request có `trace_id` là `aaaabbbbccccddddeeeeffff00001111`.
- [ ] Với một collector OTLP/gRPC ở `trace.endpoint`: trace đó gồm span server `/settings.v1.SettingsService/UpdateSetting`, các span `sql.*` là con của nó, span consumer `payment:settings.v1.SettingChanged` là con của span server, và các span `sql.*` của consumer.
- [ ] Để server chạy không tải 10 giây: collector không nhận thêm span nào.

**N. Metrics**

- [ ] `GET /metrics` có `go_sql_open_connections{db_name="main"}`, `outbox_pending_messages{db_name="main"}`, `mq_publish_time_seconds`, `mq_subscriber_messages_received_total`, `mq_handler_execution_time_seconds`.
- [ ] Dừng RabbitMQ rồi tạo vài setting: `outbox_pending_messages` tăng; bật lại RabbitMQ thì về 0.

**O. ListPayments**

- [ ] 29 bản ghi, `page_size=10`: trang 1 và 2 có 10 bản ghi, trang 3 có 9, không bản ghi nào trùng hay sót; mọi trang có `totalSize` 29.
- [ ] Không tham số: 20 bản ghi, mới nhất trước.
- [ ] `filter=currency="USD" AND amount>200` và `order_by=amount desc` trả đúng tập con, đúng thứ tự, `totalSize` là số bản ghi khớp filter.
- [ ] `page=99` trả `payments` rỗng và `totalSize` đúng; `page_size=1000` trả tối đa 100.
- [ ] `filter=nope=1`, `filter=amount >`, `order_by=id; drop table` đều trả 400 `PAYMENT_LIST_INVALID` với thông báo một dòng.
- [ ] `filter=currency="x' OR '1'='1"` trả danh sách rỗng, không lỗi.
- [ ] Gọi `ListPayments` qua gRPC cho cùng kết quả; `api/openapi/openapi.yaml` có `GET /v1/payments`.

## 14. Ngoài phạm vi

Không làm trong đợt này, kể cả khi thấy tiện tay:

- Rate limit, overload protection, security headers.
- Prometheus, Grafana, Jaeger, collector hay bất kỳ dịch vụ quan sát nào trong `docker-compose.yml`.
- Lấy mẫu trace theo tỷ lệ, TLS cho OTLP, xuất metrics qua OTLP.
- API danh sách cho module `settings`; package phân trang dùng chung; phân trang cursor.
- Consumer cho hàng đợi poison; job dọn bảng `inbox`.
- CI, biến môi trường, quy ước hay arch test mới.

**Chưa được kiểm chứng khi spike**, người làm cần tự chạy: `/readyz` khi database chết, lỗi 499 với client thật, xuất trace tới collector thật (spike dùng bộ thu tự viết), và test qua testcontainers (spike chạy trên PostgreSQL và RabbitMQ cài sẵn).
