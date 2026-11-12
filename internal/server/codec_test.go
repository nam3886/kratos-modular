package server

import (
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/encoding"

	"github.com/nam3886/kratos-modular/internal/conf"
)

// Until the payment API exists these use conf messages with the same shapes.
func TestJSONCodecFollowsTheProtobufJSONMapping(t *testing.T) {
	codec := encoding.GetCodec("json")

	out, err := codec.Marshal(&conf.Server_HTTP{Addr: "a", MaxBodyBytes: 500})
	if err != nil {
		t.Fatal(err)
	}
	// 64-bit integers are strings, and unset fields are still present.
	for _, want := range []string{`"maxBodyBytes":"500"`, `"network":""`} {
		if !strings.Contains(strings.ReplaceAll(string(out), " ", ""), want) {
			t.Errorf("%s does not contain %s", out, want)
		}
	}

	// Both camelCase and proto field names are accepted, unknown fields are ignored.
	var srv conf.Server
	if err := codec.Unmarshal([]byte(`{"http":{"maxBodyBytes":"7"},"extra":true}`), &srv); err != nil {
		t.Fatal(err)
	}
	if srv.GetHttp().GetMaxBodyBytes() != 7 {
		t.Fatalf("decoded %v", &srv)
	}
}

// A request whose HTTP body maps to one message field is decoded into a
// pointer to that field, which starts out nil.
func TestJSONCodecDecodesIntoABodyField(t *testing.T) {
	var srv conf.Server
	if err := encoding.GetCodec("json").Unmarshal([]byte(`{"maxBodyBytes":"7","addr":"a"}`), &srv.Http); err != nil {
		t.Fatal(err)
	}
	if srv.GetHttp().GetMaxBodyBytes() != 7 || srv.GetHttp().GetAddr() != "a" {
		t.Fatalf("decoded %v", &srv)
	}
}

func TestJSONCodecIgnoresEmptyBody(t *testing.T) {
	m := &conf.Server_HTTP{Addr: "keep"}
	if err := encoding.GetCodec("json").Unmarshal(nil, m); err != nil {
		t.Fatal(err)
	}
	if m.GetAddr() != "keep" {
		t.Fatalf("empty body changed the message: %v", m)
	}
}
