# Spec: kratos-modular boilerplate

Nov 11, 2026 · @Nguyen Phuong Nam

## 1. Mục tiêu và phạm vi

Dựng một boilerplate Go theo kiến trúc modular monolith trên Kratos v3: một ứng dụng triển khai duy nhất, gồm nhiều module mà compiler giữ tách biệt. Module path là `github.com/nam3886/kratos-modular`.

Spec này viết cho người (hoặc AI) triển khai chưa biết gì về quá trình thiết kế. Mọi quyết định dưới đây đã được chốt và đã chạy thử trong một bản tham chiếu; đừng thay đổi chúng nếu không có lý do kỹ thuật bắt buộc. Mục 15 liệt kê các bẫy đã gặp, nên đọc trước khi viết code.

**Sản phẩm phải giao**

- Một repository build được, có đủ cấu trúc ở mục 3 và hai module mẫu `settings`, `payment`.
- Code sinh (`*.pb.go`, Ent, `wire_gen.go`, `openapi.yaml`) được commit cùng, để clone về là build được ngay.
- `go.mod` và `go.sum` hợp lệ (`go mod tidy` không đổi gì).
- Toàn bộ tiêu chí nghiệm thu ở mục 16 chạy qua.

**Tính năng**

| Nhóm | Tính năng |
| --- | --- |
| API | Proto-first; một service cho cả HTTP và gRPC; OpenAPI sinh tự động; JSON khớp OpenAPI; mã lỗi khai báo trong proto; mọi kiểm tra đầu vào (kể cả trường thiếu hoặc bằng 0) nằm ở `biz` và trả đúng reason cụ thể |
| Module | Mỗi module ba tầng `service`, `biz`, `data` sau `internal/` riêng; Wire riêng từng module; tự đăng ký HTTP, gRPC, consumer |
| Tương tác | Gọi trực tiếp một chiều qua interface sinh từ proto; chiều ngược lại bằng event |
| Dữ liệu | Ent + PostgreSQL; n database khai báo trong config, mỗi database một pool dùng chung; mỗi module một client Ent; migration có phiên bản |
| Messaging | Outbox trong transaction; forwarder sang RabbitMQ; consumer có thử lại, hàng đợi lỗi và inbox chống trùng |
| Vận hành | Metrics Prometheus tại `/metrics`; CORS; giới hạn kích thước request; dừng có kiểm soát |
| Test | `biz`, `service` với bản giả viết tay; `data` trên PostgreSQL thật; một test tích hợp chạy cả ứng dụng |
| Công cụ | Makefile, buf, golangci-lint, Dockerfile, docker-compose, README |

## 2. Công nghệ và phiên bản

Dùng đúng các phiên bản dưới đây cho thư viện trực tiếp; đây là tổ hợp đã build và chạy test qua. Thư viện gián tiếp để `go mod tidy` tự chọn.

| Thành phần | Phiên bản | Vai trò |
| --- | --- | --- |
| Go | 1.27.4 (`go 1.27.4` trong `go.mod`) | Ngôn ngữ |
| `github.com/go-kratos/kratos/v3` | v3.0.0 | Framework: HTTP, gRPC, config, vòng đời ứng dụng |
| `github.com/go-kratos/kratos/contrib/otel/v3` | v3.0.0-20261026152050-dd0fb7df1e32 | Middleware metrics, gắn trace vào log |
| `github.com/google/wire` | v0.7.0 | Dependency injection lúc build |
| `entgo.io/ent` | v0.14.6 | ORM sinh code |
| `ariga.io/atlas` | v1.3.0 | Sinh file migration từ schema Ent |
| `github.com/pressly/goose/v3` | v3.26.0 | Áp dụng migration |
| `github.com/jackc/pgx/v5` | v5.9.2 | Driver PostgreSQL của ứng dụng (tên driver `pgx`) |
| `github.com/lib/pq` | v1.12.3 | Driver dùng riêng cho trình sinh migration |
| `github.com/ThreeDotsLabs/watermill` | v1.5.3 | Router, Forwarder (outbox), middleware |
| `github.com/ThreeDotsLabs/watermill-sql/v4` | v4.1.5 | Outbox trong PostgreSQL |
| `github.com/ThreeDotsLabs/watermill-amqp/v3` | v3.1.0 | RabbitMQ |
| `github.com/google/uuid` | v1.6.0 | Kiểu UUID ở tầng `data` (Ent yêu cầu) |
| `github.com/gorilla/handlers` | v1.5.2 | CORS |
| `github.com/prometheus/client_golang` | v1.24.1 | Endpoint `/metrics` |
| `go.opentelemetry.io/otel/exporters/prometheus` | v0.69.0 | Đưa metrics OpenTelemetry sang Prometheus |
| `go.opentelemetry.io/otel/sdk/metric` | v1.47.0 | MeterProvider |
| `google.golang.org/grpc` | v1.85.0 | gRPC |
| `google.golang.org/protobuf` | v1.36.12 | Protobuf, protojson |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | v0.45.0 | PostgreSQL cho test |
| `github.com/testcontainers/testcontainers-go/modules/rabbitmq` | v0.45.0 | RabbitMQ cho test |

**Công cụ sinh code** (chạy qua `go run` hoặc `go install`, không cần cài protoc):

- `buf` (bản mới nhất)
- `google.golang.org/protobuf/cmd/protoc-gen-go` v1.36.11
- `google.golang.org/grpc/cmd/protoc-gen-go-grpc` v1.6.2
- `github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v3` (cùng commit với Kratos v3.0.0)
- `github.com/google/gnostic/cmd/protoc-gen-openapi` v0.7.1
- `golangci-lint` v2.15.0 trở lên (phải được build bằng Go 1.27)

**Dịch vụ**: PostgreSQL (image `postgres:17-alpine`; bản tham chiếu đã chạy trên PostgreSQL 16) và RabbitMQ 4 (image `rabbitmq:4-management-alpine`; đã chạy trên 4.4.0).

**Điểm xuất phát**: template chính thức `github.com/go-kratos/kratos-layout` (nhánh dùng Kratos v3). Lấy từ đó `cmd/server/main.go`, `buf.yaml`, `buf.lock`, `buf.gen.yaml`, `buf.gen.config.yaml`, khung `Makefile`, `Dockerfile`, rồi chỉnh theo spec này.

## 3. Cấu trúc thư mục

Cây dưới đây là toàn bộ file viết tay và vị trí code sinh. Không thêm thư mục nào khác.

```text
kratos-modular/
├── api/
│   ├── openapi/openapi.yaml            # sinh từ proto
│   ├── settings/v1/
│   │   ├── settings.proto              # service, message, ErrorReason
│   │   ├── events.proto                # SettingChanged
│   │   └── *.pb.go, *_grpc.pb.go, *_http.pb.go   # sinh
│   └── payment/v1/
│       ├── payment.proto
│       ├── events.proto                # PaymentCreated
│       └── *.pb.go ...                 # sinh
├── cmd/
│   ├── server/
│   │   ├── main.go                     # khởi động ứng dụng Kratos
│   │   ├── modules.go                  # danh sách module và cách chúng nối nhau
│   │   ├── wire.go                     # injector tổng (build tag wireinject)
│   │   ├── wire_gen.go                 # sinh
│   │   └── integration_test.go         # build tag integration
│   └── migrate/main.go                 # áp dụng migration cho mọi database
├── configs/config.yaml                 # nguồn cấu hình duy nhất
├── migrations/
│   ├── migrations.go                   # embed.FS: //go:embed */*.sql
│   └── main/                           # một thư mục cho mỗi database, trùng tên database
│       ├── <timestamp>_settings_init.sql
│       ├── <timestamp>_payment_init.sql
│       └── atlas.sum
├── deploy/postgres/init.sql            # CREATE DATABASE app_dev (cho docker-compose)
├── internal/
│   ├── conf/
│   │   ├── conf.proto
│   │   └── conf.pb.go                  # sinh
│   ├── server/                         # transport, không có nghiệp vụ
│   │   ├── server.go                   # ProviderSet, interface Registrar và Consumer
│   │   ├── http.go
│   │   ├── grpc.go
│   │   ├── codec.go                    # codec JSON theo protojson
│   │   ├── codec_test.go
│   │   ├── filter.go                   # CORS, giới hạn kích thước request
│   │   ├── filter_test.go
│   │   ├── metrics.go
│   │   └── messaging.go                # đăng ký consumer trước khi chạy
│   ├── platform/                       # hạ tầng dùng chung, không có nghiệp vụ
│   │   ├── db/db.go                    # mở n database, pool dùng chung
│   │   ├── mq/
│   │   │   ├── mq.go                   # forwarder, router consumer, retry, poison
│   │   │   ├── outbox.go               # Publish trong transaction, InitTables
│   │   │   └── inbox.go                # FirstTime
│   │   ├── migrate/
│   │   │   ├── migrate.go              # Up: bảng messaging + goose
│   │   │   └── gen/gen.go              # sinh file migration từ schema Ent
│   │   └── testkit/
│   │       ├── postgres.go
│   │       └── rabbitmq.go
│   └── module/
│       ├── settings/
│       │   ├── module.go               # Module, API(), RegisterHTTP, RegisterGRPC
│       │   ├── wire.go                 # injector của module (build tag wireinject)
│       │   ├── wire_gen.go             # sinh
│       │   └── internal/
│       │       ├── service/{service.go, setting.go}
│       │       ├── biz/{biz.go, setting.go, setting_test.go}
│       │       └── data/
│       │           ├── data.go         # ProviderSet, hằng database, client Ent, inTx
│       │           ├── setting.go      # repository
│       │           ├── setting_test.go
│       │           └── ent/
│       │               ├── generate.go # //go:generate ... ent generate
│       │               ├── schema/setting.go
│       │               ├── migrate/main.go   # viết tay, build tag ignore
│       │               └── ...               # code Ent sinh
│       └── payment/
│           ├── module.go               # thêm RegisterConsumers
│           ├── wire.go, wire_gen.go
│           └── internal/
│               ├── service/{service.go, payment.go, payment_test.go, setting_events.go}
│               ├── biz/{biz.go, payment.go, payment_test.go}
│               └── data/
│                   ├── data.go
│                   ├── payment.go          # repository: lưu payment + event
│                   ├── payment_test.go
│                   ├── limit_change.go     # ghi nhận event, có inbox
│                   ├── settings_client.go  # adapter gọi module settings
│                   └── ent/{generate.go, schema/{payment.go, limitchange.go}, migrate/main.go, ...}
├── .gitignore
├── .golangci.yml
├── buf.yaml, buf.lock, buf.gen.yaml, buf.gen.config.yaml
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── README.md
└── go.mod, go.sum
```

Quy mô tham chiếu của code viết tay (không tính test và code sinh): `platform` khoảng 560 dòng, `server` 270, `cmd` 190, `settings` 260, `payment` 400. Nếu bản triển khai lớn hơn nhiều, nhiều khả năng đang thêm thứ ngoài phạm vi.

## 4. Quy tắc kiến trúc

Năm quy tắc này là cốt lõi của thiết kế. Hai quy tắc đầu do compiler và Wire tự ép, không viết test hay linter nào để kiểm tra kiến trúc.

1. **Ranh giới module.** Mọi thứ dưới `internal/module/<m>/internal/` chỉ code trong `internal/module/<m>/` import được. Code bên ngoài (module khác, `cmd/`, `server/`) chỉ được import `internal/module/<m>` và `api/<m>/v1`.
2. **Gọi trực tiếp chỉ đi một chiều.** Module A gọi B thì B không gọi A. Nếu tạo vòng, Wire báo `cycle for ...` và không sinh code.
3. **Chiều ngược lại là event** qua outbox và RabbitMQ. Giao hàng ít nhất một lần, nên handler phải chống trùng bằng inbox.
4. **Transaction nằm gọn trong một phương thức repository.** Tầng `biz` không biết transaction. Không có transaction chéo module, kể cả khi chung database.
5. **Database là hạ tầng dùng chung.** Module chọn database bằng hằng `database` trong `data.go`. Nhiều module dùng chung một database thì dùng chung một pool. Không có luật nào về module nào sở hữu bảng nào.

**Đường đi của một lời gọi trực tiếp** (payment gọi settings, trong cùng process, không qua mạng):

```text
payment/biz      khai báo interface LimitReader (đúng thứ nó cần)
payment/data     settings_client.go cài LimitReader, giữ settingsv1.SettingsServiceServer
      │ gọi hàm Go
      ▼
settings/service → settings/biz → settings/data
```

Lời gọi trong process không đi qua middleware của transport (recovery, metrics, hideInternal).

**Đường đi của một event** (settings báo cho payment):

```text
settings/data  repo.Save: trong MỘT transaction ghi bảng settings + mq.Publish(tx, event)
                   │ commit
                   ▼
forwarder (nền) đọc bảng outbox → gửi RabbitMQ → chờ broker xác nhận → xóa dòng
                   ▼
RabbitMQ: exchange fanout "settings.v1.SettingChanged" → queue "settings.v1.SettingChanged_payment"
                   ▼
payment/service  setting_events.go: giải mã proto → biz → data.Record:
                 trong MỘT transaction: mq.FirstTime(tx, ...) rồi ghi bảng limit_changes
```

**Vị trí code theo hướng dữ liệu**: thứ đi vào module (request HTTP/gRPC, event nhận được) nằm ở `service/`. Thứ đi ra (database, phát event, gọi module khác) nằm ở `data/`. `biz/` không import Ent, Watermill hay package của module khác; nó được import `api/<m>/v1` của chính mình để lấy mã lỗi.

## 5. Cấu hình

`configs/config.yaml` là nguồn cấu hình duy nhất. Không có file `.env`, không đọc biến môi trường: trong `main.go` chỉ dùng `file.NewSource(flagconf)`, bỏ `env.NewSource` của kratos-layout. Mỗi môi trường mount một bản file riêng.

**`internal/conf/conf.proto`**

```protobuf
syntax = "proto3";

package kratos.api;

import "google/protobuf/duration.proto";

option go_package = "github.com/nam3886/kratos-modular/internal/conf;conf";

message Bootstrap {
  Server server = 1;
  Data data = 2;
}

message Server {
  message HTTP {
    string network = 1;
    string addr = 2;
    google.protobuf.Duration timeout = 3;
    // Origins allowed to call the API from a browser. Empty disables CORS.
    repeated string cors_allowed_origins = 4;
    // Largest accepted request body in bytes. Zero means 1 MiB.
    int64 max_body_bytes = 5;
  }
  message GRPC {
    string network = 1;
    string addr = 2;
    google.protobuf.Duration timeout = 3;
  }
  HTTP http = 1;
  GRPC grpc = 2;
}

message Data {
  message Database {
    // Driver name registered with database/sql, e.g. "pgx".
    string driver = 1;
    // Driver-specific DSN.
    string source = 2;
    // Log every generated statement. Development only.
    bool debug = 3;
    int32 max_open_conns = 4;
  }
  message RabbitMQ {
    string url = 1;
  }
  // Every database the application connects to, by name.
  map<string, Database> databases = 1;
  RabbitMQ rabbitmq = 2;
}
```

**`configs/config.yaml`** (khớp với `docker-compose.yml`)

```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 5s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 5s
data:
  databases:
    main:
      driver: pgx
      source: postgres://app:app@127.0.0.1:5432/app?sslmode=disable
      max_open_conns: 20
  rabbitmq:
    url: amqp://guest:guest@127.0.0.1:5672/
```

Không có cờ tự tạo bảng (`auto_migrate`). Bảng chỉ được tạo bằng lệnh `migrate` (mục 12). Thêm database thứ hai là thêm một khóa dưới `data.databases` và một thư mục `migrations/<tên>/`.

## 6. API và proto

Mỗi module có hai file proto trong `api/<module>/v1/`: một file cho service, message và mã lỗi; một file `events.proto` cho event. `go_package` có dạng `github.com/nam3886/kratos-modular/api/<module>/v1;v1`.

**`api/settings/v1/settings.proto`**

```protobuf
syntax = "proto3";

package settings.v1;

import "google/api/annotations.proto";
import "google/api/field_behavior.proto";

option go_package = "github.com/nam3886/kratos-modular/api/settings/v1;v1";

// SettingsService owns the per-currency payment limits.
service SettingsService {
  rpc GetSetting (GetSettingRequest) returns (Setting) {
    option (google.api.http) = {get: "/v1/settings/{currency}"};
  }
  rpc UpdateSetting (UpdateSettingRequest) returns (Setting) {
    option (google.api.http) = {
      put: "/v1/settings/{setting.currency}"
      body: "setting"
    };
  }
}

message Setting {
  string currency = 1 [(google.api.field_behavior) = REQUIRED];
  int64 max_amount = 2;
}

message GetSettingRequest {
  string currency = 1 [(google.api.field_behavior) = REQUIRED];
}

message UpdateSettingRequest {
  Setting setting = 1 [(google.api.field_behavior) = REQUIRED];
}

enum ErrorReason {
  ERROR_REASON_UNSPECIFIED = 0;
  SETTING_NOT_FOUND = 1;
  SETTING_CURRENCY_INVALID = 2;
  SETTING_MAX_AMOUNT_INVALID = 3;
}
```

**`api/settings/v1/events.proto`**

```protobuf
syntax = "proto3";

package settings.v1;

option go_package = "github.com/nam3886/kratos-modular/api/settings/v1;v1";

// SettingChanged is published after a setting is created or updated.
message SettingChanged {
  string currency = 1;
  int64 max_amount = 2;
}
```

**`api/payment/v1/payment.proto`**

```protobuf
syntax = "proto3";

package payment.v1;

import "google/api/annotations.proto";
import "google/api/field_behavior.proto";

option go_package = "github.com/nam3886/kratos-modular/api/payment/v1;v1";

service PaymentService {
  rpc CreatePayment (CreatePaymentRequest) returns (Payment) {
    option (google.api.http) = {
      post: "/v1/payments"
      body: "payment"
    };
  }
  rpc GetPayment (GetPaymentRequest) returns (Payment) {
    option (google.api.http) = {get: "/v1/payments/{id}"};
  }
}

message Payment {
  string id = 1 [(google.api.field_behavior) = OUTPUT_ONLY];
  int64 amount = 2 [(google.api.field_behavior) = REQUIRED];
  string currency = 3 [(google.api.field_behavior) = REQUIRED];
  string status = 4 [(google.api.field_behavior) = OUTPUT_ONLY];
}

message CreatePaymentRequest {
  Payment payment = 1 [(google.api.field_behavior) = REQUIRED];
}

message GetPaymentRequest {
  string id = 1 [(google.api.field_behavior) = REQUIRED];
}

enum ErrorReason {
  ERROR_REASON_UNSPECIFIED = 0;
  PAYMENT_NOT_FOUND = 1;
  PAYMENT_AMOUNT_INVALID = 2;
  PAYMENT_LIMIT_EXCEEDED = 3;
  PAYMENT_CURRENCY_INVALID = 4;
  PAYMENT_ID_INVALID = 5;
  PAYMENT_CURRENCY_NOT_SUPPORTED = 6;
}
```

**`api/payment/v1/events.proto`**

```protobuf
syntax = "proto3";

package payment.v1;

option go_package = "github.com/nam3886/kratos-modular/api/payment/v1;v1";

// PaymentCreated is published after a payment is stored.
message PaymentCreated {
  string id = 1;
  int64 amount = 2;
  string currency = 3;
}
```

**Quy ước**

- **Tên topic của event** là tên proto đầy đủ của message, ví dụ `settings.v1.SettingChanged`. Payload là message proto mã hóa nhị phân.
- **Lỗi** tạo bằng `errors.NotFound(v1.ErrorReason_X.String(), "...")` hoặc `errors.BadRequest(...)` của `github.com/go-kratos/kratos/v3/errors`, khai báo thành biến trong `biz`.
- **JSON** theo ánh xạ JSON của protobuf (mục 8): tên trường camelCase, `int64` là chuỗi. Ví dụ body tạo payment: `{"amount":"500","currency":"usd"}`; body cập nhật setting: `{"currency":"USD","maxAmount":"1000"}`.
- **OpenAPI** sinh ra một file `api/openapi/openapi.yaml` cho mọi module (plugin `protoc-gen-openapi`, tham số `fq_schema_naming=true`, `default_response=false`, `strategy: all`).

## 7. Hạ tầng dùng chung (`internal/platform`)

Bốn package, không package nào chứa nghiệp vụ hay import module. Chiều phụ thuộc: `db` ← `mq` ← `migrate` ← `testkit`.

### 7.1 `platform/db`

Mở mỗi database trong config đúng một lần và giữ pool cho cả ứng dụng.

```go
type Database struct {
	Name  string
	Pool  *sql.DB
	Debug bool
}

type Manager struct{ /* map[string]*Database */ }

// Open mở một pool cho mỗi database và Ping từng cái. cleanup đóng mọi pool.
func Open(c *conf.Data) (*Manager, func(), error)

// Get trả database theo tên; lỗi nếu tên không có trong config.
func (m *Manager) Get(name string) (*Database, error)

// All trả mọi database.
func (m *Manager) All() []*Database
```

- Dùng `sql.Open(driver, source)`, gọi `SetMaxOpenConns` khi `max_open_conns > 0`.
- Import trống `github.com/jackc/pgx/v5/stdlib` để đăng ký driver `pgx`.
- Package này không biết Ent. Mỗi module tự tạo client Ent trên pool (mục 9).

### 7.2 `platform/mq`

Lớp messaging dựng trên Watermill. Ba file.

**`outbox.go`** — ghi event trong transaction và tạo bảng:

```go
// Execer là một transaction đang chạy: *sql.Tx hoặc *ent.Tx (sinh với cờ sql/execquery).
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// executor chuyển Execer thành wsql.ContextExecutor (kiểu trả về khác nhau nên cần bọc).
type executor struct{ tx Execer }

func (e executor) ExecContext(ctx context.Context, q string, args ...any) (wsql.Result, error) {
	return e.tx.ExecContext(ctx, q, args...)
}

func (e executor) QueryContext(ctx context.Context, q string, args ...any) (wsql.Rows, error) {
	return e.tx.QueryContext(ctx, q, args...)
}

func outboxSchema() wsql.PostgreSQLQueueSchema { return wsql.PostgreSQLQueueSchema{} }

// Publish lưu event vào outbox qua tx. Event chỉ tới RabbitMQ nếu tx commit.
func Publish(ctx context.Context, tx Execer, id string, event proto.Message) error {
	payload, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	sqlPub, err := wsql.NewPublisher(executor{tx}, wsql.PublisherConfig{SchemaAdapter: outboxSchema()}, watermill.NopLogger{})
	if err != nil {
		return err
	}
	pub := forwarder.NewPublisher(sqlPub, forwarder.PublisherConfig{ForwarderTopic: outboxTopic})
	msg := message.NewMessage(id, payload)
	msg.SetContext(ctx)
	return pub.Publish(Topic(event), msg)
}

// Topic = tên proto đầy đủ của event.
func Topic(event proto.Message) string {
	return string(event.ProtoReflect().Descriptor().FullName())
}

// InitTables tạo bảng outbox (các câu lệnh của outboxSchema().SchemaInitializingQueries
// với Topic = outboxTopic) và bảng inbox trong một database. Chạy ngoài transaction, lặp lại an toàn.
func InitTables(ctx context.Context, pool *sql.DB) error
```

Hằng: `outboxTopic = "outbox"` (bảng sẽ tên `watermill_outbox`), `PoisonTopic = "poison"`. Bảng inbox:

```sql
CREATE TABLE IF NOT EXISTS inbox (
    consumer   TEXT NOT NULL,
    message_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (consumer, message_id)
)
```

**`inbox.go`**

```go
// FirstTime ghi nhận, trong tx, rằng consumer đã xử lý message và báo đây có phải lần đầu không.
// Message gửi lại trả false; handler khi đó không làm gì.
func FirstTime(ctx context.Context, tx Execer, consumer, messageID string) (bool, error)
// INSERT INTO inbox (consumer, message_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
// trả RowsAffected() == 1
```

**`mq.go`** — forwarder và consumer:

```go
// Handler xử lý một message. Trả lỗi thì thử lại.
type Handler func(ctx context.Context, msg *message.Message) error

// MQ cài transport.Server của Kratos (Start/Stop) để ứng dụng điều khiển vòng đời.
type MQ struct{ /* url, logger, publisher, router, forwarders, subs, cancel, wg */ }

func New(c *conf.Data, dbs *db.Manager, log *slog.Logger) (*MQ, func(), error)
func (m *MQ) Handle(consumer, topic string, h Handler) error
func (m *MQ) Start(ctx context.Context) error
func (m *MQ) Stop(context.Context) error
```

Hành vi bắt buộc:

- **Cấu hình AMQP**: `amqp.NewDurablePubSubConfig(url, amqp.GenerateQueueNameTopicNameWithSuffix(consumer))` và đặt `cfg.Publish.ConfirmDelivery = true`. Kết quả: mỗi topic một exchange fanout bền, mỗi consumer một queue bền tên `<topic>_<consumer>`.
- **`New`**: tạo một `amqp.Publisher`; tạo `message.Router` với middleware theo thứ tự `PoisonQueueWithFilter(pub, PoisonTopic, err => !errors.Is(err, context.Canceled))` (chỉ `Canceled`, không phải `DeadlineExceeded`; message bị ngắt do shutdown được broker giao lại thay vì vào poison), `Retry{MaxRetries: 3, InitialInterval: 100ms, Multiplier: 2}`, `Recoverer`; với **mỗi** database tạo một forwarder: `wsql.NewSubscriber(wsql.BeginnerFromStdSQL(pool), SubscriberConfig{SchemaAdapter: outboxSchema(), OffsetsAdapter: wsql.PostgreSQLQueueOffsetsAdapter{DeleteOnAck: true}, PollInterval: 200ms})` rồi `forwarder.NewForwarder(sub, pub, logger, forwarder.Config{ForwarderTopic: outboxTopic})`. `New` không tạo bảng, nhưng khai báo queue bền `poison_poison` gắn với exchange `poison` (tạo `amqp.Subscriber` với `config(url, "poison")`, `SubscribeInitialize(PoisonTopic)`, rồi đóng). Logger là `watermill.NewSlogLogger(log)`. cleanup đóng publisher.
- **`Handle`**: tạo một `amqp.Subscriber` riêng, gọi ngay `sub.SubscribeInitialize(topic)` để khai báo queue trước khi ứng dụng nhận request, rồi `router.AddConsumerHandler(consumer+":"+topic, topic, sub, ...)`.
- **`Start`**: chạy mọi `forwarder.Run(ctx)` và `router.Run(ctx)` trong goroutine, chặn cho tới khi tất cả kết thúc. Context dẫn xuất bằng `context.WithCancel(context.WithoutCancel(ctx))`. Lỗi khác `context.Canceled` thì hủy tất cả và trả về lỗi gộp.
- **`Stop`**: hủy context, `Close` từng forwarder và router, chờ goroutine, rồi đóng các subscriber.

### 7.3 `platform/migrate`

```go
// Up đưa mọi database lên phiên bản mới nhất. Với từng database:
//  1. mq.InitTables(ctx, pool)
//  2. dir := fs.Sub(migrations.FS, d.Name)
//  3. goose.NewProvider(goose.DialectPostgres, pool, dir); goose.ErrNoMigrations thì bỏ qua
//  4. p.Up(ctx), log từng file đã áp dụng
func Up(ctx context.Context, dbs *db.Manager, log *slog.Logger) error
```

`platform/migrate/gen` chứa `func Main(tables []*schema.Table)` cho trình sinh migration; chi tiết ở mục 12. Tách package riêng để binary server không kéo Atlas và `lib/pq`.

### 7.4 `platform/testkit`

```go
// PostgresURL trả DSN của PostgreSQL test. Nếu biến TEST_DATABASE_URL có giá trị thì dùng nó,
// nếu không thì khởi động container postgres:17-alpine một lần cho mỗi test binary (sync.Once).
func PostgresURL(t testing.TB) string

// Postgres trả Manager có database "main" trống và đã migrate.
func Postgres(t testing.TB) *db.Manager

// ResetPostgres: DROP SCHEMA public CASCADE; CREATE SCHEMA public; rồi migrate.Up.
func ResetPostgres(t testing.TB, dbs *db.Manager)

// Count đếm số dòng của một bảng trong database "main".
func Count(t testing.TB, dbs *db.Manager, table string) int

// RabbitMQURL: TEST_RABBITMQ_URL nếu có, nếu không thì container rabbitmq:4-alpine.
func RabbitMQURL(t testing.TB) string
```

- Container: `postgres.Run(ctx, "postgres:17-alpine", postgres.WithDatabase("test"), postgres.WithUsername("test"), postgres.WithPassword("test"), postgres.BasicWaitStrategies())` rồi `ctr.ConnectionString(ctx, "sslmode=disable")`; `rabbitmq.Run(ctx, "rabbitmq:4-alpine")` rồi `ctr.AmqpURL(ctx)`.
- Test dùng chính các file migration thật, không dùng cơ chế tạo bảng nào khác.
- Hai biến môi trường trên chỉ phục vụ test, không phải cấu hình ứng dụng.

## 8. Server (`internal/server`)

Package này dựng transport và không chứa nghiệp vụ. Nó không import module nào; module tự đăng ký qua hai interface.

**`server.go`**

```go
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer, NewMessaging, NewMetrics)

// Registrar là module có API.
type Registrar interface {
	RegisterHTTP(*http.Server)   // github.com/go-kratos/kratos/v3/transport/http
	RegisterGRPC(*grpc.Server)   // github.com/go-kratos/kratos/v3/transport/grpc
}

// Consumer là module có xử lý event.
type Consumer interface {
	RegisterConsumers(*mq.MQ) error
}
```

**`http.go`** — `func NewHTTPServer(c *conf.Server, m *Metrics, modules []Registrar) *http.Server`

- Filter (chạy trước khi giải mã): luôn có `bodyLimit(c.GetHttp().GetMaxBodyBytes())`; thêm `cors(c.GetHttp().GetCorsAllowedOrigins())` khi danh sách khác rỗng. Truyền bằng `http.Filter(filters...)`.
- Middleware theo thứ tự: `recovery.Recovery()`, `m.middleware` (metrics), `hideInternal`. Không có validator ở tầng transport: `biz` kiểm tra và trả reason cụ thể.
- Áp `Network`, `Address`, `Timeout` từ config khi có giá trị.
- **`errors.go`**: `hideInternal` là middleware dùng chung cho HTTP và gRPC: lỗi không phải `*errors.Error` của Kratos được ghi log đầy đủ (`slog.ErrorContext`) rồi trả `errors.InternalServer("INTERNAL", "internal error")`; lỗi Kratos đi qua nguyên vẹn. Có một test đơn vị.
- Sau `http.NewServer(opts...)`: `srv.Handle("/metrics", m.handler)`, rồi gọi `RegisterHTTP` của từng module.

**`grpc.go`** — `func NewGRPCServer(c *conf.Server, m *Metrics, modules []Registrar) *grpc.Server`: middleware `recovery.Recovery()`, `m.middleware`, `hideInternal`; áp config; gọi `RegisterGRPC` của từng module.

**`messaging.go`**

```go
// Messaging là server messaging: forwarder và consumer.
type Messaging struct{ *mq.MQ }

// NewMessaging đăng ký mọi module consumer trước khi bất cứ thứ gì chạy.
func NewMessaging(q *mq.MQ, consumers []Consumer) (*Messaging, error)
```

**`codec.go`** — bắt buộc, chép nguyên. Codec JSON mặc định của Kratos v3 dùng `encoding/json` trên struct sinh, cho ra tên trường snake\_case, timestamp dạng `{"seconds":..,"nanos":..}` và enum dạng số, không khớp OpenAPI. Codec này đăng ký đè lên tên `"json"`:

```go
package server

import (
	stdjson "encoding/json"
	"reflect"

	"github.com/go-kratos/kratos/v3/encoding"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type jsonCodec struct{}

var (
	marshal   = protojson.MarshalOptions{EmitUnpopulated: true}
	unmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}
)

func init() { encoding.RegisterCodec(jsonCodec{}) }

func (jsonCodec) Name() string { return "json" }

func (jsonCodec) Marshal(v any) ([]byte, error) {
	if m, ok := v.(proto.Message); ok {
		return marshal.Marshal(m)
	}
	return stdjson.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	if m, ok := message(v); ok {
		return unmarshal.Unmarshal(data, m)
	}
	return stdjson.Unmarshal(data, v)
}

// message trả v dưới dạng proto message. Body gắn vào một trường message
// (ví dụ body: "payment") đến dưới dạng con trỏ tới trường đó, có thể đang nil.
func message(v any) (proto.Message, bool) {
	if m, ok := v.(proto.Message); ok {
		return m, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() || rv.Elem().Kind() != reflect.Pointer {
		return nil, false
	}
	field := rv.Elem()
	if field.IsNil() {
		field.Set(reflect.New(field.Type().Elem()))
	}
	m, ok := field.Interface().(proto.Message)
	return m, ok
}
```

**`filter.go`**

```go
const defaultMaxBodyBytes = 1 << 20

// cors cho phép trình duyệt ở các origin đã khai báo gọi API.
func cors(origins []string) http.FilterFunc {
	return handlers.CORS(
		handlers.AllowedOrigins(origins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)
}

// bodyLimit từ chối body lớn hơn limit byte (limit <= 0 thì dùng mặc định 1 MiB).
func bodyLimit(limit int64) http.FilterFunc
```

Hành vi `bodyLimit`: nếu `r.ContentLength > limit` trả ngay HTTP 413 với body `{"code":413,"reason":"REQUEST_TOO_LARGE","message":"request body is too large"}` và `Content-Type: application/json`. Ngược lại bọc `r.Body = http.MaxBytesReader(w, r.Body, limit)` rồi chuyển tiếp; body không khai báo độ dài mà vượt giới hạn sẽ thành lỗi giải mã 400 của Kratos.

**`metrics.go`**

```go
type Metrics struct {
	middleware middleware.Middleware
	handler    nethttp.Handler
}

func NewMetrics() (*Metrics, error)
```

- Tạo `prometheus.NewRegistry()` riêng (không dùng registry toàn cục, để test tạo ứng dụng nhiều lần không lỗi trùng), đăng ký `collectors.NewGoCollector()` và `collectors.NewProcessCollector(...)`.
- `otelprom.New(otelprom.WithRegisterer(registry))` làm reader cho `sdkmetric.NewMeterProvider`, kèm view `metrics.DefaultSecondsHistogramView(metrics.DefaultServerSecondsHistogramName)`.
- Meter tên `"server"`; tạo `metrics.DefaultRequestsCounter(meter, metrics.DefaultServerRequestsCounterName)` và `metrics.DefaultSecondsHistogram(meter, metrics.DefaultServerSecondsHistogramName)`.
- `middleware = metrics.Server(metrics.WithRequests(requests), metrics.WithSeconds(seconds))`; `handler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})`.
- Package `metrics` là `github.com/go-kratos/kratos/contrib/otel/v3/metrics` (Kratos v3 không còn `middleware/metrics` trong lõi).
- Kết quả mong đợi tại `GET /metrics`: `server_requests_code_total{kind,operation,code,reason}`, `server_requests_seconds_*`, cùng `go_*` và `process_*`.

Tracing không nằm trong phạm vi. Giữ nguyên phần log của kratos-layout trong `main.go` (gắn `trace.id`, `span.id` vào log qua `tracing.TraceAttrs`).

## 9. Cấu trúc một module

Mỗi module là một kratos-layout thu nhỏ. Ví dụ dưới đây lấy từ `settings`; `payment` theo cùng khuôn. Dùng `M` thay cho `github.com/nam3886/kratos-modular`.

**`module.go`** (package tên theo module) — thứ duy nhất bên ngoài được dùng:

```go
// Package settings is the payment-settings module. Other modules may use only
// this package and api/settings/v1.
package settings

type Module struct {
	svc *service.SettingsService
}

func newModule(svc *service.SettingsService) *Module { return &Module{svc: svc} }

// API là thứ module khác gọi, trong process.
func (m *Module) API() v1.SettingsServiceServer { return m.svc }

func (m *Module) RegisterHTTP(srv *http.Server) { v1.RegisterSettingsServiceHTTPServer(srv, m.svc) }
func (m *Module) RegisterGRPC(srv *grpc.Server) { v1.RegisterSettingsServiceServer(srv, m.svc) }
```

- Chỉ module bị module khác gọi mới có `API()`.
- Module có consumer thêm `RegisterConsumers(q *mq.MQ) error`.

**`wire.go`** — bắt buộc nằm trong module, vì `cmd/server` không import được các package `internal` của module:

```go
//go:build wireinject

package settings

// New assembles the module.
func New(*db.Manager) (*Module, error) {
	panic(wire.Build(data.ProviderSet, biz.ProviderSet, service.ProviderSet, newModule))
}
```

Tham số của `New` là mọi thứ module cần từ bên ngoài: `*db.Manager`, và interface API của module nó gọi (với `payment`: `settingsv1.SettingsServiceServer`).

**Ba tầng**, mỗi tầng có một file cùng tên chứa `ProviderSet`:

| File | Nội dung |
| --- | --- |
| `internal/service/service.go` | `var ProviderSet = wire.NewSet(NewSettingsService)` |
| `internal/service/<x>.go` | Struct nhúng `v1.Unimplemented...Server`, giữ usecase; chỉ chuyển đổi proto ↔ kiểu `biz` |
| `internal/biz/biz.go` | `var ProviderSet = wire.NewSet(NewSettingUsecase)` |
| `internal/biz/<x>.go` | Entity, biến lỗi, interface repository và interface tới module khác, usecase |
| `internal/data/data.go` | `ProviderSet`, hằng `database`, struct `Data`, `NewData`, `inTx` |
| `internal/data/<x>.go` | Repository cài interface của `biz`; constructor trả kiểu interface của `biz` |

**`internal/data/data.go`**

```go
// database là tên database (trong config) mà module này lưu vào.
const database = "main"

var ProviderSet = wire.NewSet(NewData, NewSettingRepo)

// Data giữ client Ent của module. Client chạy trên pool dùng chung nên KHÔNG được Close ở đây.
type Data struct {
	db *ent.Client
}

func NewData(dbs *db.Manager) (*Data, error) {
	d, err := dbs.Get(database)
	if err != nil {
		return nil, err
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, d.Pool)))
	if d.Debug {
		client = client.Debug()
	}
	return &Data{db: client}, nil
}

// inTx chạy fn trong một transaction: commit khi fn trả nil, rollback khi lỗi hoặc panic.
func (d *Data) inTx(ctx context.Context, fn func(tx *ent.Tx) error) error {
	tx, err := d.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()
	if err := fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return errors.Join(err, fmt.Errorf("rollback: %w", rerr))
		}
		return err
	}
	return tx.Commit()
}
```

`inTx` là toàn bộ cơ chế transaction. Không truyền transaction qua `context`, không có hook sau commit, không có package transaction dùng chung. `ent.Tx` là kiểu sinh riêng của từng module nên mỗi module có bản `inTx` của mình.

**Ent**

- Schema nằm ở `internal/data/ent/schema/`.
- `internal/data/ent/generate.go`:

```go
package ent

//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/execquery,sql/upsert ./schema
```

- Cờ `sql/execquery` bắt buộc: nó thêm `ExecContext`/`QueryContext` vào `*ent.Tx`, nhờ đó `*ent.Tx` thỏa `mq.Execer`. Cờ `sql/upsert` thêm `OnConflict*` cho `Save` của settings.
- `internal/data/ent/migrate/main.go` viết tay (mục 12); `ent generate` không xóa file này.

## 10. Hai module mẫu

`settings` giữ hạn mức thanh toán theo loại tiền. `payment` tạo thanh toán, gọi `settings` để kiểm tra hạn mức và nhận event khi hạn mức đổi. `payment` phụ thuộc `settings`; chiều ngược lại không tồn tại.

### 10.1 Module `settings`

**Schema Ent** `Setting`: `field.String("id").StorageKey("currency")` (khóa chính là mã tiền), `field.Int64("max_amount")`. Bảng: `settings(currency varchar PK, max_amount bigint)`.

**`biz`**

```go
var (
	ErrSettingNotFound        = errors.NotFound(v1.ErrorReason_SETTING_NOT_FOUND.String(), "setting not found")
	ErrSettingCurrencyInvalid  = errors.BadRequest(v1.ErrorReason_SETTING_CURRENCY_INVALID.String(), "currency must be a 3-letter code")
	ErrSettingMaxAmountInvalid = errors.BadRequest(v1.ErrorReason_SETTING_MAX_AMOUNT_INVALID.String(), "max amount must be positive")
)

type Setting struct {
	Currency  string
	MaxAmount int64
}

// SettingRepo lưu setting. Save cũng phát event thay đổi.
type SettingRepo interface {
	Find(ctx context.Context, currency string) (*Setting, error)
	Save(ctx context.Context, s *Setting) (*Setting, error)
}

func NewSettingUsecase(repo SettingRepo) *SettingUsecase
func (uc *SettingUsecase) Get(ctx context.Context, currency string) (*Setting, error)
func (uc *SettingUsecase) Update(ctx context.Context, s *Setting) (*Setting, error)
```

- `Get`: viết hoa mã tiền rồi `repo.Find`.
- `Update`: viết hoa mã tiền; nếu độ dài khác 3 trả `ErrSettingCurrencyInvalid`; nếu `MaxAmount <= 0` trả `ErrSettingMaxAmountInvalid`; ngược lại `repo.Save`.

**`data`**

- `Find`: `ent.IsNotFound(err)` thì trả `biz.ErrSettingNotFound`.
- `Save`: trong một `inTx`, upsert một câu lệnh (`INSERT ... ON CONFLICT (id) DO UPDATE`), rồi phát event; trả lại giá trị vừa lưu:

```go
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
```

**`service`**: `GetSetting` và `UpdateSetting` chỉ chuyển đổi giữa proto và `biz.Setting`.

### 10.2 Module `payment`

**Schema Ent**

- `Payment`: `field.UUID("id", uuid.UUID{})` (kiểu của `github.com/google/uuid`), `field.Int64("amount")`, `field.String("currency")`, `field.String("status").Default("pending")`.
- `LimitChange`: `field.String("currency")`, `field.Int64("max_amount")`, `field.Time("created_at").Default(time.Now).Immutable()`. Bảng `limit_changes`, khóa chính `id` tự tăng.

**`biz`**

```go
var (
	ErrPaymentNotFound        = errors.NotFound(v1.ErrorReason_PAYMENT_NOT_FOUND.String(), "payment not found")
	ErrPaymentAmountInvalid   = errors.BadRequest(v1.ErrorReason_PAYMENT_AMOUNT_INVALID.String(), "amount must be positive")
	ErrPaymentCurrencyInvalid = errors.BadRequest(v1.ErrorReason_PAYMENT_CURRENCY_INVALID.String(), "currency must be a 3-letter code")
	ErrPaymentIDInvalid       = errors.BadRequest(v1.ErrorReason_PAYMENT_ID_INVALID.String(), "payment id must be a UUID")
	ErrCurrencyNotSupported   = errors.BadRequest(v1.ErrorReason_PAYMENT_CURRENCY_NOT_SUPPORTED.String(), "currency is not supported")
	ErrLimitExceeded          = errors.BadRequest(v1.ErrorReason_PAYMENT_LIMIT_EXCEEDED.String(), "amount exceeds the currency limit")
)

type Payment struct {
	ID       uuid.UUID // gói "uuid" của thư viện chuẩn Go 1.27
	Amount   int64
	Currency string
	Status   string
}

// PaymentRepo lưu payment. Create cũng phát event PaymentCreated.
type PaymentRepo interface {
	Create(ctx context.Context, p *Payment) (*Payment, error)
	Find(ctx context.Context, id uuid.UUID) (*Payment, error)
}

// LimitReader là thứ module này cần từ module settings.
type LimitReader interface {
	MaxAmount(ctx context.Context, currency string) (int64, error)
}

// LimitChangeRepo ghi nhận thay đổi hạn mức mà module được báo.
type LimitChangeRepo interface {
	Record(ctx context.Context, messageID, currency string, maxAmount int64) error
}

func NewPaymentUsecase(repo PaymentRepo, limits LimitReader, changes LimitChangeRepo) *PaymentUsecase
func (uc *PaymentUsecase) Create(ctx context.Context, amount int64, currency string) (*Payment, error)
func (uc *PaymentUsecase) Get(ctx context.Context, id uuid.UUID) (*Payment, error)
func (uc *PaymentUsecase) LimitChanged(ctx context.Context, messageID, currency string, maxAmount int64) error
```

Luật của `Create`, theo thứ tự:

1. Viết hoa mã tiền. `amount <= 0` → `ErrPaymentAmountInvalid`; độ dài mã khác 3 → `ErrPaymentCurrencyInvalid`.
2. `limits.MaxAmount(ctx, currency)`; lỗi thì trả nguyên lỗi đó.
3. `amount > limit` → `ErrLimitExceeded` (bằng đúng hạn mức vẫn hợp lệ).
4. `repo.Create` với `ID: uuid.NewV7()`.

**`data`**

- `payment.go` — `Create` trong một `inTx`: `tx.Payment.Create().SetID(guuid.UUID(p.ID))...Save(ctx)` rồi `mq.Publish(ctx, tx, uuid.NewV7().String(), &v1.PaymentCreated{Id: p.ID.String(), Amount: p.Amount, Currency: p.Currency})`. Chuyển giữa hai kiểu UUID bằng ép kiểu trực tiếp `guuid.UUID(id)` và `uuid.UUID(row.ID)`. `Find`: không thấy thì trả `biz.ErrPaymentNotFound`.
- `settings_client.go` — `NewLimitReader(settings settingsv1.SettingsServiceServer) biz.LimitReader`. `MaxAmount` gọi `settings.GetSetting(ctx, &settingsv1.GetSettingRequest{Currency: currency})`; `errors.IsNotFound(err)` → `biz.ErrCurrencyNotSupported`; lỗi khác trả nguyên.
- `limit_change.go` — hằng xuất `Consumer = "payment"` khai báo trong `data.go` cạnh `database` (là hậu tố queue AMQP và khóa inbox; đổi tên sẽ giao lại và xử lý lại message cũ):

```go
func (r *limitChangeRepo) Record(ctx context.Context, messageID, currency string, maxAmount int64) error {
	return r.data.inTx(ctx, func(tx *ent.Tx) error {
		first, err := mq.FirstTime(ctx, tx, Consumer, messageID)
		if err != nil || !first {
			return err
		}
		return tx.LimitChange.Create().SetCurrency(currency).SetMaxAmount(maxAmount).Exec(ctx)
	})
}
```

- `ProviderSet = wire.NewSet(NewData, NewPaymentRepo, NewLimitChangeRepo, NewLimitReader)`.

**`service`**

- `payment.go` — `CreatePayment` gọi `uc.Create(ctx, req.GetPayment().GetAmount(), req.GetPayment().GetCurrency())`. `GetPayment` phân tích id bằng `uuid.Parse`; lỗi → `biz.ErrPaymentIDInvalid`.
- `setting_events.go` — handler event:

```go
type SettingEvents struct{ uc *biz.PaymentUsecase }

func NewSettingEvents(uc *biz.PaymentUsecase) *SettingEvents

// OnSettingChanged xử lý settings.v1.SettingChanged.
func (h *SettingEvents) OnSettingChanged(ctx context.Context, msg *message.Message) error {
	var event settingsv1.SettingChanged
	if err := proto.Unmarshal(msg.Payload, &event); err != nil {
		return err
	}
	return h.uc.LimitChanged(ctx, msg.UUID, event.GetCurrency(), event.GetMaxAmount())
}
```

- `ProviderSet = wire.NewSet(NewPaymentService, NewSettingEvents)`.

**`module.go`**: `Module` giữ `svc` và `events`; có `RegisterHTTP`, `RegisterGRPC` và:

```go
func (m *Module) RegisterConsumers(q *mq.MQ) error {
	return q.Handle(data.Consumer, mq.Topic(&settingsv1.SettingChanged{}), m.events.OnSettingChanged)
}
```

**`wire.go`**: `func New(*db.Manager, settingsv1.SettingsServiceServer) (*Module, error)`.

Event `PaymentCreated` được phát nhưng không module nào trong repo nhận; nó minh họa chiều phát cho hệ thống bên ngoài.

## 11. Khởi động và wiring

**`cmd/server/main.go`** — lấy từ kratos-layout với ba thay đổi:

- Bỏ `env.NewSource("KRATOS")`, chỉ còn `file.NewSource(flagconf)`.
- Bỏ import `go.uber.org/automaxprocs` (Go 1.25 trở lên tự xử lý).
- `newApp` nhận thêm `*server.Messaging` và đưa vào `kratos.Server(gs, hs, ms)`.

```go
func newApp(logger *slog.Logger, gs *grpc.Server, hs *http.Server, ms *server.Messaging) *kratos.App
```

Cờ dòng lệnh: `-conf` (mặc định `../../configs`). `defer func() { _ = c.Close() }()` cho config.

**`cmd/server/modules.go`** — nơi duy nhất liệt kê module:

```go
package main

// settingsAPI là thứ module payment được trao để gọi settings.
func settingsAPI(s *settings.Module) settingsv1.SettingsServiceServer {
	return s.API()
}

// registrars liệt kê module có API.
func registrars(s *settings.Module, p *payment.Module) []server.Registrar {
	return []server.Registrar{s, p}
}

// consumers liệt kê module có xử lý event.
func consumers(p *payment.Module) []server.Consumer {
	return []server.Consumer{p}
}
```

**`cmd/server/wire.go`**

```go
//go:build wireinject

package main

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		db.Open,
		mq.New,
		settings.New,
		settingsAPI,
		payment.New,
		registrars,
		consumers,
		server.ProviderSet,
		newApp,
	))
}
```

Thứ tự khởi tạo Wire sinh ra: `db.Open` → `settings.New` → `settingsAPI` → `payment.New` → `registrars` → `NewGRPCServer`/`NewHTTPServer` → `mq.New` → `consumers` → `NewMessaging` → `newApp`. Queue của consumer được khai báo trong `NewMessaging`, tức là trước `app.Run()`.

**`cmd/migrate/main.go`** — cờ `-conf` như server. Đọc config bằng `config.New(config.WithSource(file.NewSource(path)))`, `Scan` vào `conf.Bootstrap`, `db.Open(bc.Data)`, rồi `migrate.Up(ctx, dbs, log)`. Thoát mã 1 khi lỗi. Logger là `slog.NewTextHandler(os.Stdout, nil)`.

**Thêm một module mới** là bốn việc (ghi vào README):

1. Viết `api/<tên>/v1/*.proto`, chạy `make api`.
2. Chép `internal/module/settings` (hoặc `payment` nếu có consumer) thành `internal/module/<tên>` và thay nội dung; module có consumer phải đổi `data.Consumer` (trong `data.go`) thành tên của module.
3. Thêm module vào `cmd/server/wire.go` và `cmd/server/modules.go`, chạy `make ent wire`.
4. `make migration module=<tên> name=<tên>_init`, rồi `make migrate`.

## 12. Migration

Bảng nghiệp vụ chỉ được tạo và đổi bằng file SQL có phiên bản. File được sinh từ schema Ent (Atlas, định dạng goose) và áp dụng bằng goose. Ứng dụng, test và production dùng chung một đường này.

**Bố cục**

- `migrations/<tên database>/*.sql`: một thư mục cho mỗi database trong config, **không** chia theo module. Các module chung database ghi file vào chung thư mục.
- `migrations/migrations.go`:

```go
// Package migrations embeds the SQL migrations: one directory per configured
// database, named after it.
package migrations

import "embed"

// FS holds every database's migration files.
//
//go:embed */*.sql
var FS embed.FS
```

- File SQL được nhúng vào binary; `migrate` không cần thư mục `migrations/` lúc chạy.
- `migrations/<db>/atlas.sum` do trình sinh ghi; commit cùng.

**Trình sinh** — `internal/platform/migrate/gen/gen.go`:

```go
// Main phân tích dòng lệnh và ghi migration cho tables.
func Main(tables []*schema.Table) {
	name := flag.String("name", "", "migration name, e.g. add_payment_note")
	database := flag.String("db", "main", "configured database the module stores into")
	devURL := flag.String("dev-url", "postgres://app:app@127.0.0.1:5432/app_dev?sslmode=disable",
		"an empty scratch database used to replay the existing migrations")
	flag.Parse()
	// -name rỗng: in lỗi, thoát mã 2. Lỗi khi sinh: in lỗi, thoát mã 1.
	run(context.Background(), *devURL, filepath.Join("migrations", *database), *name, tables)
}

func run(ctx context.Context, devURL, path, name string, tables []*schema.Table) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	dir, err := sqltool.NewGooseDir(path)
	if err != nil {
		return err
	}
	// Chấp nhận file migration viết hoặc sửa tay: làm mới file checksum
	// thay vì từ chối chạy khi nó không còn khớp.
	sum, err := dir.Checksum()
	if err != nil {
		return err
	}
	if err := migrate.WriteSumFile(dir, sum); err != nil {
		return err
	}
	m, err := schema.NewMigrateURL(devURL,
		schema.WithDir(dir),
		schema.WithMigrationMode(schema.ModeReplay),
		schema.WithDialect(dialect.Postgres),
		schema.WithFormatter(sqltool.GooseFormatter),
	)
	if err != nil {
		return err
	}
	return m.NamedDiff(ctx, name, tables...)
}
```

Import: `ariga.io/atlas/sql/migrate`, `ariga.io/atlas/sql/sqltool`, `entgo.io/ent/dialect`, `entgo.io/ent/dialect/sql/schema`, và import trống `github.com/lib/pq`.

Mỗi module có `internal/data/ent/migrate/main.go`:

```go
//go:build ignore

// Command main writes a migration for this module's schema changes:
//
//	make migration module=settings name=<what_changed>
package main

import (
	"github.com/nam3886/kratos-modular/internal/module/settings/internal/data/ent/migrate"
	"github.com/nam3886/kratos-modular/internal/platform/migrate/gen"
)

func main() {
	gen.Main(migrate.Tables)
}
```

**Hành vi đã kiểm chứng của trình sinh**

- Nó phát lại toàn bộ file trong thư mục lên database trống `app_dev`, so với schema Ent của module, rồi ghi một file `<timestamp>_<name>.sql`. Database `app_dev` được trả về trạng thái trống sau khi chạy.
- Nó chỉ xét bảng của module đang chạy: sinh migration cho `payment` không tạo lệnh xóa bảng `settings`.
- Không có thay đổi thì không ghi file nào.
- File viết tay theo định dạng goose (`-- +goose Up` / `-- +goose Down`) đặt chung thư mục được chấp nhận.

File sinh ra có dạng:

```sql
-- +goose Up
-- create "settings" table
CREATE TABLE "settings" ("currency" character varying NOT NULL, "max_amount" bigint NOT NULL, PRIMARY KEY ("currency"));

-- +goose Down
-- reverse: create "settings" table
DROP TABLE "settings";
```

**Áp dụng** — `platform/migrate.Up` (mục 7.3), gọi từ `cmd/migrate` và từ `testkit`. Nó cũng tạo bảng outbox và inbox, nên server lúc chạy không cần quyền tạo bảng. Chạy hai lần liên tiếp phải an toàn (lần hai áp dụng 0 file).

**Quy trình đổi schema**

1. Sửa schema Ent của module.
2. `make ent`
3. `make migration module=<module> name=<mô_tả>` (thêm `db=<tên>` nếu module không dùng `main`)
4. `make migrate`

**Bản giao ban đầu** phải chứa hai file sinh bằng chính quy trình này: `<timestamp>_settings_init.sql` (bảng `settings`) và `<timestamp>_payment_init.sql` (bảng `limit_changes` và `payments`), cùng `atlas.sum`.

## 13. Test

Dùng gì thì test bằng cái đó: test tầng `data` và test tích hợp chạy trên PostgreSQL và RabbitMQ thật. Không dùng SQLite, không dùng thư viện sinh mock, không có test luật kiến trúc.

| Tầng | Cách test | Phụ thuộc |
| --- | --- | --- |
| `biz` | Go thuần, bản giả viết tay của interface repository và `LimitReader` | Không |
| `service` | Gọi thẳng phương thức service với usecase thật + repository giả | Không |
| `data` | Repository thật trên `testkit.Postgres(t)` | PostgreSQL |
| `server` | Unit test cho codec và `bodyLimit` | Không |
| `cmd/server` (build tag `integration`) | Chạy cả ứng dụng qua `wireApp` | PostgreSQL, RabbitMQ |

**Danh sách test bắt buộc**

`internal/module/settings/internal/biz/setting_test.go`

- `TestUpdateThenGet`: cập nhật `usd`/1000, đọc lại bằng `Usd` được `USD`/1000; đọc `EUR` trả `ErrSettingNotFound`.
- `TestUpdateRejectsInvalidSettings`: mã `US`, hạn mức 0, hạn mức âm trả `ErrSettingCurrencyInvalid` (mã `US`) hoặc `ErrSettingMaxAmountInvalid` (hạn mức 0 và âm).

`internal/module/settings/internal/data/setting_test.go`

- `TestSaveStoresSettingAndEventTogether`: `Find` khi chưa có trả `ErrSettingNotFound`; `Save` hai lần cùng mã tiền (1000 rồi 2000) cho ra 1 dòng `settings` giá trị 2000 và 2 dòng `watermill_outbox`.

`internal/module/payment/internal/biz/payment_test.go`

- `TestCreate` dạng bảng, hạn mức USD = 1000: 500/`usd` hợp lệ; 1000 hợp lệ; 1001 → `ErrLimitExceeded`; `EUR` → `ErrCurrencyNotSupported`; số tiền 0 → `ErrPaymentAmountInvalid`; mã `US` → `ErrPaymentCurrencyInvalid`. Trường hợp lỗi không được lưu gì; trường hợp hợp lệ có mã tiền viết hoa và ID khác `uuid.Nil()`.

`internal/module/payment/internal/service/payment_test.go`

- `TestCreateThenGet`: tạo rồi đọc lại, mã tiền `USD`, trạng thái `pending`.
- `TestErrorsCarryTheAPIReason`: vượt hạn mức có `errors.Reason(err) == "PAYMENT_LIMIT_EXCEEDED"` và `errors.Code(err) == 400`; id không phải UUID có reason `PAYMENT_ID_INVALID`; id không tồn tại thỏa `errors.IsNotFound`.

`internal/module/payment/internal/data/payment_test.go`

- `TestCreateStoresPaymentAndEventTogether`: sau `Create` có 1 dòng `payments` và 1 dòng `watermill_outbox`, trạng thái `pending`; `Create` lại cùng ID phải lỗi và hai bảng vẫn 1 và 1 (transaction rollback cả event); `Find` đúng; `Find` ID lạ trả `ErrPaymentNotFound`.
- `TestRecordIsIdempotentPerMessage`: `Record` ba lần với `msg-1` và một lần với `msg-2` cho đúng 2 dòng `limit_changes`.

`internal/server/codec_test.go`

- `TestJSONCodecFollowsTheProtobufJSONMapping`: `encoding.GetCodec("json").Marshal(&paymentv1.Payment{Id:"p-1", Amount:500, Currency:"USD"})` chứa `"amount":"500"` và `"status":""` (bỏ khoảng trắng trước khi so, vì protojson chèn khoảng trắng ngẫu nhiên); giải mã `{"payment":{"amount":"7","currency":"USD"},"extra":true}` vào `CreatePaymentRequest` được amount 7.
- `TestJSONCodecDecodesIntoABodyField`: giải mã `{"amount":"7","currency":"USD"}` vào `&req.Payment` (con trỏ tới trường đang nil) được amount 7, currency `USD`.

`internal/server/filter_test.go`

- `TestBodyLimit` với giới hạn 10 byte: body 10 byte → 200; 11 byte có khai báo độ dài → 413; 11 byte với `ContentLength = -1` → handler đọc lỗi (400).
- `TestBodyLimitDefaultsToOneMebibyte`: `bodyLimit(0)` từ chối request khai báo `1<<20 + 1` byte.

`cmd/server/integration_test.go` (`//go:build integration`, package `main`)

- `TestPaymentFlow`: `testkit.Postgres(t)`; dựng `conf.Bootstrap` với HTTP `127.0.0.1:18000`, gRPC `127.0.0.1:19000`, database `main` = `testkit.PostgresURL(t)`, RabbitMQ = `testkit.RabbitMQURL(t)`; `wireApp(...)` với logger bỏ đi; `go app.Run()`, cuối test `app.Stop()` và kiểm tra `Run` trả nil. Qua HTTP: `PUT /v1/settings/usd` → 200; `POST /v1/payments` 500 USD → 200, `status == "pending"`; 5000 USD → 400, `reason == "PAYMENT_LIMIT_EXCEEDED"`; `GET /v1/payments/{id}` → 200. Sau đó chờ tối đa 15 giây cho tới khi `limit_changes` có 1 dòng và `watermill_outbox` có 0 dòng.

**Chạy test**

- `go test ./...` và `go test -tags integration ./...` tự khởi động container; mỗi package test có container riêng nên chạy song song được.
- Khi đặt `TEST_DATABASE_URL` và `TEST_RABBITMQ_URL`, các package dùng chung một database nên phải thêm `-p 1`.

## 14. Công cụ

**`Makefile`** — giữ target `help` của kratos-layout làm mặc định; mỗi target có một dòng chú thích `# ...` phía trên. Biến: `module ?=`, `name ?=`, `db ?= main`.

| Target | Lệnh |
| --- | --- |
| `init` | `go install` cho `github.com/google/wire/cmd/wire@latest`, `github.com/bufbuild/buf/cmd/buf@latest`, `github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` |
| `api` | `buf generate --template buf.gen.yaml` |
| `config` | `buf generate --template buf.gen.config.yaml` |
| `ent` | `go generate ./internal/module/*/internal/data/ent` |
| `wire` | `go run -mod=mod github.com/google/wire/cmd/wire ./internal/module/... ./cmd/...` |
| `generate` | `api config ent wire`, rồi `go mod tidy` |
| `build` | `mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./cmd/...` |
| `up` | `docker compose up -d --wait` |
| `down` | `docker compose down` |
| `migration` | Kiểm tra `module` và `name` khác rỗng (thiếu thì in cách dùng, thoát mã 2), rồi `go run -mod=mod ./internal/module/$(module)/internal/data/ent/migrate/main.go -name $(name) -db $(db)` |
| `migrate` | `go run ./cmd/migrate -conf ./configs` |
| `run` | `go run ./cmd/server -conf ./configs` |
| `test` | `go test ./...` |
| `test-integration` | `go test -tags integration ./...` |
| `lint` | `golangci-lint run ./...` |

`VERSION=$(shell git describe --tags --always 2>/dev/null || echo dev)`.

**buf** — chép `buf.yaml`, `buf.lock`, `buf.gen.config.yaml` nguyên từ kratos-layout. `buf.gen.yaml` cũng chép từ đó (input `directory: api`; plugin `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-go-http/v3`, `protoc-gen-openapi`, tất cả dạng `local: ["go", "run", "<pkg>@<version>"]`), chỉ đổi `out` của plugin openapi thành `api/openapi`. Ba plugin đầu có `out: api`, `opt: paths=source_relative`.

**`.golangci.yml`**

```yaml
version: "2"

run:
  build-tags:
    - integration

linters:
  # The standard set: errcheck, govet, ineffassign, staticcheck, unused.
  default: standard
  enable:
    - bodyclose
    - errorlint
    - misspell
    - nilerr
    - unconvert
  exclusions:
    # Generated code (*.pb.go, ent, wire_gen.go) is recognised by its header.
    generated: strict

formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - github.com/nam3886/kratos-modular
```

Không có CI; chỉ có `make lint`.

**`Dockerfile`** — hai giai đoạn: `golang:1.27` chạy `make build`; `debian:stable-slim` cài `ca-certificates`, `netbase`, chép `/src/bin` vào `/app`. `EXPOSE 8000 9000`, `VOLUME /data/conf`, `CMD ["./server", "-conf", "/data/conf"]`. Không đặt `GOPROXY`. Ghi chú rằng `./migrate -conf /data/conf` chạy một lần mỗi bản phát hành, trước khi khởi động server.

**`docker-compose.yml`**

```yaml
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
      POSTGRES_DB: app
    ports:
      - "5432:5432"
    volumes:
      - postgres:/var/lib/postgresql/data
      - ./deploy/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d app"]
      interval: 2s
      timeout: 3s
      retries: 30

  rabbitmq:
    image: rabbitmq:4-management-alpine
    ports:
      - "5672:5672"
      - "15672:15672"
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "-q", "ping"]
      interval: 5s
      timeout: 5s
      retries: 30

volumes:
  postgres:
```

`deploy/postgres/init.sql` chỉ có `CREATE DATABASE app_dev;` (database trống cho `make migration`).

**`.gitignore`**: `bin/`, `*.exe`, `*.test`, `*.out`, `vendor/`, `go.work`, `go.work.sum`, `.idea/`, `.vscode/`, `.DS_Store`.

**`README.md`** (tiếng Anh, ngắn) gồm các mục: giới thiệu một đoạn; Requirements; Quick start (`go mod tidy`, `make init`, `make up`, `make migrate`, `make run`, ba lệnh `curl` mẫu); Layout; How modules relate (năm quy tắc ở mục 4); Adding a module; Changing the schema; Tests; Configuration; Operations (chạy `migrate` trước server, queue `poison_poison` có sẵn để replay, queue consumer khai báo lúc khởi động, quy ước JSON).

## 15. Bẫy đã biết

Mỗi mục dưới đây là một lỗi đã gặp thật khi dựng bản tham chiếu, kèm cách đúng.

| # | Bẫy | Cách đúng |
| --- | --- | --- |
| 1 | Nối `data`, `biz`, `service` của module ngay trong `cmd/server/wire.go`: code sinh import package `internal` của module và không biên dịch | Mỗi module có `wire.go` và `wire_gen.go` riêng, xuất hàm `New` |
| 2 | Truyền method value `(*settings.Module).API` vào `wire.Build`: Wire báo "unknown pattern" | Viết hàm thường `settingsAPI` trong `modules.go` |
| 3 | JSON mặc định của Kratos v3 không khớp OpenAPI | Đăng ký `jsonCodec` ở mục 8 |
| 4 | Codec chỉ xử lý `proto.Message`: request có `body: "<trường>"` được giải mã vào con trỏ tới trường đó, rơi sang `encoding/json` và mất dữ liệu | Hàm `message()` trong codec xử lý con trỏ tới con trỏ |
| 5 | Thiếu cờ `sql/execquery` khi sinh Ent: `*ent.Tx` không có `ExecContext`/`QueryContext` | Luôn sinh với `--feature sql/execquery` |
| 6 | `wsql.ContextExecutor` trả `wsql.Result`/`wsql.Rows`, còn `*ent.Tx` trả `sql.Result`/`*sql.Rows`: không gán trực tiếp được | Bọc bằng struct `executor` |
| 7 | `Close()` client Ent của một module đóng luôn pool dùng chung | Không đóng client; chỉ `db.Open` đóng pool |
| 8 | `field.UUID` của Ent đòi `driver.Valuer`, kiểu `uuid.UUID` của thư viện chuẩn không có | Schema dùng `github.com/google/uuid`; `biz` dùng `uuid` chuẩn; ép kiểu ở `data` |
| 9 | Event gửi lên exchange fanout chưa có queue nào: RabbitMQ xác nhận rồi bỏ, dòng outbox bị xóa | `MQ.Handle` gọi `SubscribeInitialize` ngay khi đăng ký, trước khi ứng dụng chạy |
| 10 | Forwarder gửi theo lô và chỉ xóa khi xong lô: dừng hoặc sập giữa lô làm gửi lại | Mọi handler dùng `mq.FirstTime` trong cùng transaction với thay đổi của nó |
| 11 | Message lỗi chuyển sang exchange `poison` chưa có queue thì mất | `New` khai báo queue bền `poison_poison`; vận hành replay từ đó |
| 12 | Tạo bảng outbox trong `mq.New`: test tầng `data` buộc phải có RabbitMQ | `InitTables` là hàm riêng, gọi từ `migrate.Up` |
| 13 | `//go:embed */*.sql` lỗi biên dịch khi chưa có file `.sql` nào | Sinh migration đầu tiên trước khi build `migrations` |
| 14 | Trình sinh migration từ chối chạy khi `atlas.sum` không khớp (do file viết tay) | Ghi lại checksum trước khi so sánh (mục 12) |
| 15 | Trình sinh migration cần driver `postgres` của `lib/pq`, ứng dụng dùng `pgx` | Import trống `github.com/lib/pq` trong package `gen` |
| 16 | Dùng registry Prometheus toàn cục: tạo ứng dụng lần hai trong test báo đăng ký trùng | `prometheus.NewRegistry()` riêng cho mỗi `Metrics` |
| 17 | Kratos v3 không có `middleware/metrics` trong lõi | Dùng `contrib/otel/v3/metrics` |
| 18 | protojson chèn khoảng trắng ngẫu nhiên vào JSON | Test không so chuỗi JSON nguyên văn; bỏ khoảng trắng hoặc giải mã rồi so |
| 19 | `int64` trong JSON là chuỗi (`"amount":"500"`) | Đúng chuẩn, không sửa; ghi trong README |
| 20 | Hai module đặt trùng tên bảng trong cùng database | Không có công cụ chặn; đặt tên bảng khác nhau |
| 21 | `golangci-lint` build bằng Go cũ hơn 1.27 từ chối phân tích | Dùng bản v2.15.0 trở lên |
| 22 | Lỗi errcheck và errorlint phổ biến | `defer func() { _ = c.Close() }()`; ghép lỗi bằng `errors.Join`, không `%v` |
| 23 | Router HTTP của Kratos rơi về `http.DefaultServeMux` khi không khớp route, nơi thư viện Prometheus đăng ký `/debug/vars` (expvar) | Đặt `NotFoundHandler` và `MethodNotAllowedHandler` riêng ở `NewHTTPServer`; `GET /debug/vars` phải trả 404 |

**Điều bản tham chiếu chưa kiểm chứng** (người triển khai phải tự chạy):

- `go mod tidy` với proxy Go thông thường và việc sinh `go.sum`.
- Nhánh testcontainers của `testkit` (đã biên dịch, chưa chạy vì không có Docker).
- `docker-compose.yml` và `Dockerfile`.
- `make api` và `make config` qua buf (code đã sinh bằng `protoc` với cùng plugin và tham số).

## 16. Tiêu chí nghiệm thu

Bản triển khai chỉ được coi là xong khi mọi mục dưới đây đã chạy thật và đạt. Báo cáo kết quả từng mục; mục nào không chạy được phải nói rõ, không suy đoán.

**A. Build và kiểm tra tĩnh**



- [ ] `go mod tidy` không đổi `go.mod`, `go.sum`
- [ ] `go build ./...` và `go vet -tags integration ./...` không lỗi
- [ ] `make lint` báo 0 lỗi
- [ ] `make generate` chạy lại không tạo ra thay đổi nào trong git

**B. Ranh giới** (thử rồi hoàn nguyên)



- [ ] Thêm import `internal/module/settings/internal/biz` vào một file của `payment`: biên dịch lỗi "use of internal package ... not allowed"
- [ ] Thêm import tương tự vào `cmd/server`: cùng lỗi
- [ ] Cho `settings` nhận `paymentv1.PaymentServiceServer`: `wire ./cmd/server` báo "cycle for ..."

**C. Migration**



- [ ] `make up` rồi `make migrate` trên database trống: áp dụng 2 file; có các bảng `settings`, `payments`, `limit_changes`, `watermill_outbox`, `inbox`, `goose_db_version`
- [ ] `make migrate` lần hai: áp dụng 0 file
- [ ] `make migration module=payment name=noop` khi schema không đổi: không tạo file
- [ ] Thêm một trường vào schema `Payment`, `make ent`, `make migration module=payment name=add_note`: tạo một file chỉ có `ALTER TABLE "payments" ADD COLUMN ...`, không đụng bảng `settings`
- [ ] Đặt một file goose viết tay vào `migrations/main/` rồi chạy `make migration`: không báo lỗi checksum

**D. Luồng chính** (`make run`)



- [ ] `PUT /v1/settings/usd` với `{"currency":"USD","maxAmount":"1000"}` → 200, body có `"maxAmount":"1000"`
- [ ] Cùng request với `{"currency":"USD","max_amount":2000}` → 200 (chấp nhận tên proto và số)
- [ ] `POST /v1/payments` với `{"amount":"500","currency":"usd"}` → 200, `currency` là `USD`, `status` là `pending`
- [ ] `POST /v1/payments` số tiền 5000 → 400, `reason` là `PAYMENT_LIMIT_EXCEEDED`
- [ ] `POST /v1/payments` loại tiền chưa có setting → 400, message "currency is not supported"
- [ ] `POST /v1/payments` thiếu `amount` → 400, `reason` là `PAYMENT_AMOUNT_INVALID`; thiếu `currency` → `PAYMENT_CURRENCY_INVALID`
- [ ] `GET /v1/payments/{id}` → 200
- [ ] Gọi `PaymentService.CreatePayment` qua gRPC cổng 9000 → thành công; vượt hạn mức trả `InvalidArgument`
- [ ] Sau mỗi lần cập nhật setting, bảng `limit_changes` và `inbox` tăng 1 dòng trong vài giây, `watermill_outbox` về 0
- [ ] RabbitMQ có exchange `settings.v1.SettingChanged`, `payment.v1.PaymentCreated` và queue `settings.v1.SettingChanged_payment`

**E. Sự cố**



- [ ] Dừng RabbitMQ, gửi 100 lần `PUT /v1/settings/usd`: cả 100 trả 200, `watermill_outbox` có 100 dòng
- [ ] Bật lại RabbitMQ, `kill -9` server khi đang chuyển tiếp, khởi động lại: `limit_changes` tăng đúng 100, `watermill_outbox` về 0
- [ ] Gửi thẳng lên exchange `settings.v1.SettingChanged` một message không phải proto hợp lệ: log có 3 lần thử lại, message xuất hiện ở queue `poison_poison`, message hợp lệ gửi sau vẫn được xử lý
- [ ] `SIGTERM` server: thoát trong vài giây, không còn kết nối PostgreSQL nào của ứng dụng

**F. Vận hành**



- [ ] Đặt `cors_allowed_origins: [https://app.example.com]`: preflight `OPTIONS` từ origin đó trả header `Access-Control-Allow-Origin`; từ origin khác không có header này
- [ ] Đặt `max_body_bytes: 2048`, gửi body 5000 byte → 413, `reason` là `REQUEST_TOO_LARGE`
- [ ] `GET /metrics` có `server_requests_code_total` với nhãn `operation`, `code`, `reason`; có `server_requests_seconds_count`, `go_goroutines`, `process_resident_memory_bytes`
- [ ] Khi rảnh, ứng dụng giữ không quá vài kết nối PostgreSQL (một pool cho mọi module), không phải một pool mỗi module

**G. Nhiều database** (thử rồi hoàn nguyên)



- [ ] Thêm database `second` vào config, đổi hằng `database` của `payment` thành `"second"`, chuyển file migration của payment sang `migrations/second/`: bảng `payments` nằm ở database thứ hai, mỗi database có `watermill_outbox` riêng, luồng D vẫn đạt

**H. Test**



- [ ] `make test` và `make test-integration` đạt, chạy 3 lần liên tiếp đều đạt
- [ ] Có đủ các test ở mục 13

**I. Docker**



- [ ] `docker build .` thành công; image có `/app/server` và `/app/migrate`

## 17. Ngoài phạm vi

Những thứ dưới đây đã được cân nhắc và cố ý không làm. Đừng thêm chúng; mục tiêu là một boilerplate gọn.

| Không làm | Lý do |
| --- | --- |
| Module cửa (gateway/BFF) bọc API của các module khác | Mỗi tài nguyên phải khai báo hai lần; module tự đăng ký API của mình |
| Trình sinh module | Chép một module mẫu và làm theo bốn bước trong README là đủ |
| Test hoặc linter kiểm tra luật kiến trúc | Compiler và Wire đã ép hai luật quan trọng nhất |
| Luật sở hữu database hoặc bảng theo module; schema PostgreSQL riêng từng module | Database là hạ tầng dùng chung |
| Tự tạo bảng lúc khởi động (`Schema.Create`) | Chỉ một đường tạo bảng: migration |
| Transaction truyền qua `context`, hook sau commit, chế độ mặc định chỉ đọc, transaction chéo module | `inTx` trong một phương thức repository là đủ; outbox thay cho hook sau commit |
| Gọi gRPC qua mạng giữa các module | Cùng process, gọi hàm Go qua interface sinh từ proto |
| Biến môi trường, file `.env` | Chỉ `config.yaml` |
| CI | Chỉ `make lint` |
| Xác thực, module user, JWT | Phụ thuộc hệ thống của từng dự án |
| Tracing (exporter, middleware) | Chỉ có metrics và gắn trace id vào log |
| Rate limit, chống quá tải, security header, client IP, sanitize | Thêm khi có nhu cầu |
| Redis, cache, HTTP client dùng chung | Không module mẫu nào cần |
| API upload/download file | Đã kiểm chứng làm được bằng `google.api.HttpBody`, nhưng không đưa vào boilerplate |
| Cron job | Chưa có job cụ thể; cần quyết định cơ chế khóa khi chạy nhiều bản |
| Tách command/query, repository in-memory, kho event, seeder | Không cần ở quy mô module mẫu |
| Sinh mock (Mockery, gomock) | Bản giả viết tay ngắn hơn |
| Validation bằng protovalidate hoặc validator ở transport | Kiểm tra trong `biz` là đủ và cho reason cụ thể; `field_behavior = REQUIRED` chỉ để tài liệu OpenAPI |
| Swagger UI | File `openapi.yaml` mở được bằng công cụ ngoài |
| Package dịch lỗi database dùng chung | Dùng `ent.IsNotFound`, `ent.IsConstraintError` tại repository |
| File ghi quyết định kiến trúc, thư mục `docs/` | README đã ghi các quy tắc |
| OpenBao, SigNoz, Grafana, gom log, Kubernetes manifest | Hạ tầng ngoài repo |
