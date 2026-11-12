package server

import (
	stdjson "encoding/json"
	"reflect"

	"github.com/go-kratos/kratos/v3/encoding"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// jsonCodec replaces Kratos' default "json" codec so that API messages are
// written the way the generated OpenAPI document describes them: camelCase
// names, RFC 3339 timestamps, enums as names. Kratos' default uses
// encoding/json on the generated structs, which does none of that.
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

// message returns v as a proto message. A request body bound to a message
// field arrives as a pointer to that (possibly nil) field.
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
