//go:build integration

package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/platform/testkit"

	"google.golang.org/protobuf/types/known/durationpb"
)

// TestPaymentFlow runs the whole application against real PostgreSQL and
// RabbitMQ: HTTP in, a call between modules, and an event from one module
// handled by the other.
func TestPaymentFlow(t *testing.T) {
	dbs := testkit.Postgres(t)
	main, err := dbs.Get("main")
	if err != nil {
		t.Fatal(err)
	}
	pool := main.Pool

	bc := &conf.Bootstrap{
		Server: &conf.Server{
			Http: &conf.Server_HTTP{Addr: "127.0.0.1:18000", Timeout: durationpb.New(5 * time.Second)},
			Grpc: &conf.Server_GRPC{Addr: "127.0.0.1:19000", Timeout: durationpb.New(5 * time.Second)},
		},
		Data: &conf.Data{
			Databases: map[string]*conf.Data_Database{"main": {Driver: "pgx", Source: testkit.PostgresURL(t)}},
			Rabbitmq:  &conf.Data_RabbitMQ{Url: testkit.RabbitMQURL(t)},
		},
	}
	app, cleanup, err := wireApp(bc.Server, bc.Data, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	done := make(chan error, 1)
	go func() { done <- app.Run() }()
	defer func() {
		if err := app.Stop(); err != nil {
			t.Errorf("stop: %v", err)
		}
		if err := <-done; err != nil {
			t.Errorf("run: %v", err)
		}
	}()

	const base = "http://127.0.0.1:18000"
	call := func(method, path, body string) (int, map[string]any) {
		t.Helper()
		req, _ := http.NewRequest(method, base+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		var res *http.Response
		for range 50 {
			if res, err = http.DefaultClient.Do(req); err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
			req, _ = http.NewRequest(method, base+path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		}
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer func() { _ = res.Body.Close() }()
		out := map[string]any{}
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}

	if code, _ := call("PUT", "/v1/settings/usd", `{"max_amount":1000}`); code != 200 {
		t.Fatalf("update setting: %d", code)
	}
	code, payment := call("POST", "/v1/payments", `{"amount":500,"currency":"usd"}`)
	if code != 200 || payment["status"] != "pending" {
		t.Fatalf("create payment: %d %v", code, payment)
	}
	if code, body := call("POST", "/v1/payments", `{"amount":5000,"currency":"usd"}`); code != 400 || body["reason"] != "PAYMENT_LIMIT_EXCEEDED" {
		t.Fatalf("over limit: %d %v", code, body)
	}
	if code, got := call("GET", "/v1/payments/"+payment["id"].(string), ""); code != 200 || got["id"] != payment["id"] {
		t.Fatalf("get payment: %d %v", code, got)
	}

	// The settings module's event must reach the payment module through the
	// outbox and RabbitMQ, and both outbox rows must be forwarded.
	deadline := time.Now().Add(15 * time.Second)
	for {
		var changes, outbox int
		_ = pool.QueryRow("SELECT count(*) FROM limit_changes").Scan(&changes)
		_ = pool.QueryRow("SELECT count(*) FROM watermill_outbox").Scan(&outbox)
		if changes == 1 && outbox == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("limit_changes=%d outbox=%d after 15s", changes, outbox)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
