package server

import (
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/encoding"

	paymentv1 "github.com/nam3886/kratos-modular/api/payment/v1"
)

func TestJSONCodecFollowsTheProtobufJSONMapping(t *testing.T) {
	codec := encoding.GetCodec("json")

	out, err := codec.Marshal(&paymentv1.Payment{Id: "p-1", Amount: 500, Currency: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	// 64-bit integers are strings, and unset fields are still present.
	for _, want := range []string{`"amount":"500"`, `"status":""`} {
		if !strings.Contains(strings.ReplaceAll(string(out), " ", ""), want) {
			t.Errorf("%s does not contain %s", out, want)
		}
	}

	// Both camelCase and proto field names are accepted, unknown fields are ignored.
	var req paymentv1.CreatePaymentRequest
	if err := codec.Unmarshal([]byte(`{"payment":{"amount":"7","currency":"USD"},"extra":true}`), &req); err != nil {
		t.Fatal(err)
	}
	if req.GetPayment().GetAmount() != 7 {
		t.Fatalf("decoded %v", &req)
	}
}

// A request whose HTTP body maps to one message field is decoded into a
// pointer to that field, which starts out nil.
func TestJSONCodecDecodesIntoABodyField(t *testing.T) {
	var req paymentv1.CreatePaymentRequest
	if err := encoding.GetCodec("json").Unmarshal([]byte(`{"amount":"7","currency":"USD"}`), &req.Payment); err != nil {
		t.Fatal(err)
	}
	if req.GetPayment().GetAmount() != 7 || req.GetPayment().GetCurrency() != "USD" {
		t.Fatalf("decoded %v", &req)
	}
}

func TestJSONCodecIgnoresEmptyBody(t *testing.T) {
	m := &paymentv1.Payment{Currency: "keep"}
	if err := encoding.GetCodec("json").Unmarshal(nil, m); err != nil {
		t.Fatal(err)
	}
	if m.GetCurrency() != "keep" {
		t.Fatalf("empty body changed the message: %v", m)
	}
}
