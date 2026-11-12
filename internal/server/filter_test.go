package server

import (
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimit(t *testing.T) {
	read := nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			w.WriteHeader(nethttp.StatusBadRequest)
		}
	})
	h := bodyLimit(10)(read)

	tests := []struct {
		name          string
		body          string
		declareLength bool
		want          int
	}{
		{"within the limit", "0123456789", true, nethttp.StatusOK},
		{"declared too large", "0123456789x", true, nethttp.StatusRequestEntityTooLarge},
		{"too large without a declared length", "0123456789x", false, nethttp.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			if !tt.declareLength {
				req.ContentLength = -1
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestBodyLimitDefaultsToOneMebibyte(t *testing.T) {
	for _, limit := range []int64{0, -1} {
		h := bodyLimit(limit)(nethttp.HandlerFunc(func(nethttp.ResponseWriter, *nethttp.Request) {}))
		for length, want := range map[int64]int{defaultMaxBodyBytes + 1: nethttp.StatusRequestEntityTooLarge, defaultMaxBodyBytes: nethttp.StatusOK} {
			req := httptest.NewRequest("POST", "/", strings.NewReader("x"))
			req.ContentLength = length
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != want {
				t.Fatalf("limit %d, length %d: status = %d, want %d", limit, length, rec.Code, want)
			}
		}
	}
}
