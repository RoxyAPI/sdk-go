package roxyapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	roxyapi "github.com/RoxyAPI/sdk-go"
)

// newTestServer serves one canned response and records the last request, so the
// client tests run offline and assert exactly what goes on the wire.
func newTestServer(t *testing.T, status int, body string) (*httptest.Server, *http.Request) {
	t.Helper()
	last := &http.Request{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*last = *r.Clone(context.Background())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, last
}

func TestNewRoxyRequiresKey(t *testing.T) {
	if _, err := roxyapi.NewRoxy(""); err == nil {
		t.Fatal("NewRoxy with an empty key returned no error")
	}
}

func TestRequestShape(t *testing.T) {
	srv, last := newTestServer(t, 200, `[]`)
	roxy, err := roxyapi.NewRoxy("sk_test_key", roxyapi.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewRoxy: %v", err)
	}
	if _, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil); err != nil {
		t.Fatalf("ListZodiacSigns: %v", err)
	}
	if got, want := last.URL.Path, "/astrology/signs"; got != want {
		t.Errorf("path = %q, want %q (base URL from WithBaseURL)", got, want)
	}
	if got := last.Header.Get("X-API-Key"); got != "sk_test_key" {
		t.Errorf("X-API-Key = %q, want sk_test_key", got)
	}
	if got, want := last.Header.Get("X-SDK-Client"), "roxy-sdk-go/"+roxyapi.Version; got != want {
		t.Errorf("X-SDK-Client = %q, want %q", got, want)
	}
}

func TestErrorMapping(t *testing.T) {
	srv, _ := newTestServer(t, 401, `{"error":"Invalid API key","code":"invalid_api_key"}`)
	roxy, err := roxyapi.NewRoxy("sk_bad", roxyapi.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewRoxy: %v", err)
	}
	_, err = roxy.Astrology.ListZodiacSigns(context.Background(), nil)
	var re *roxyapi.RoxyError
	if !errors.As(err, &re) {
		t.Fatalf("error = %v (%T), want *RoxyError", err, err)
	}
	if re.StatusCode != 401 || re.Code != "invalid_api_key" || re.Message != "Invalid API key" {
		t.Errorf("RoxyError = %+v, want 401 invalid_api_key Invalid API key", *re)
	}
	if got, want := re.Error(), "roxyapi: 401 invalid_api_key: Invalid API key"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// specOperation is the slice of one operation the surface tests read.
type specOperation struct {
	OperationID string `json:"operationId"`
	RequestBody struct {
		Content map[string]struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Content map[string]struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
}

type specDoc struct {
	Paths      map[string]map[string]specOperation `json:"paths"`
	Components struct {
		Schemas map[string]json.RawMessage `json:"schemas"`
	} `json:"components"`
}

func loadSpec(t *testing.T) specDoc {
	t.Helper()
	raw, err := os.ReadFile("specs/openapi.json")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var doc specDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	return doc
}

// fold compares identifiers across naming schemes: "vedic-astrology" and
// "VedicAstrology" fold to the same key, as do an operationId and its Go method.
func fold(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// facadeMethod resolves a spec operation to its method on the service of its first
// path segment, or fails the test naming what is missing.
func facadeMethod(t *testing.T, roxy *roxyapi.Roxy, path, verb string, op specOperation) (reflect.Method, bool) {
	t.Helper()
	segment := strings.Split(strings.Trim(path, "/"), "/")[0]
	rv := reflect.ValueOf(roxy).Elem()
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Type().Field(i)
		if !f.IsExported() || fold(f.Name) != fold(segment) {
			continue
		}
		for m := 0; m < f.Type.NumMethod(); m++ {
			if fold(f.Type.Method(m).Name) == fold(op.OperationID) {
				return f.Type.Method(m), true
			}
		}
		t.Errorf("%s %s: operationId %q has no method on roxy.%s", strings.ToUpper(verb), path, op.OperationID, f.Name)
		return reflect.Method{}, false
	}
	t.Errorf("%s: no service on Roxy for path segment %q", path, segment)
	return reflect.Method{}, false
}

// TestFacadeCoversSpec walks every operation of the committed spec and asserts the
// facade exposes it on the service of its first path segment, that every service is
// wired by NewRoxy, and that the facade carries nothing the spec does not.
func TestFacadeCoversSpec(t *testing.T) {
	doc := loadSpec(t)
	roxy, err := roxyapi.NewRoxy("sk_test_key")
	if err != nil {
		t.Fatalf("NewRoxy: %v", err)
	}
	seen := map[string]bool{}
	ops := 0
	for path, item := range doc.Paths {
		for verb, op := range item {
			ops++
			if m, ok := facadeMethod(t, roxy, path, verb, op); ok {
				seen[m.Type.In(0).String()+"."+m.Name] = true
			}
		}
	}
	methods := 0
	rv := reflect.ValueOf(roxy).Elem()
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Type().Field(i)
		if !f.IsExported() {
			continue
		}
		if rv.Field(i).IsNil() {
			t.Errorf("roxy.%s is nil after NewRoxy", f.Name)
		}
		for m := 0; m < f.Type.NumMethod(); m++ {
			methods++
			if name := f.Type.String() + "." + f.Type.Method(m).Name; !seen[name] {
				t.Errorf("%s is on the facade but matches no operation of the spec", name)
			}
		}
	}
	if ops == 0 || methods != ops {
		t.Errorf("%d facade methods for %d spec operations", methods, ops)
	}
}

// TestNullableFieldsArePointers pairs every request body and 200 response of the
// spec with the Go type the facade method takes or returns, and asserts that each
// field the spec declares nullable (`type: [x, "null"]`) is a pointer in Go, so a
// null on the wire can never read as a zero value.
func TestNullableFieldsArePointers(t *testing.T) {
	doc := loadSpec(t)
	roxy, err := roxyapi.NewRoxy("sk_test_key")
	if err != nil {
		t.Fatalf("NewRoxy: %v", err)
	}
	checked := 0
	for path, item := range doc.Paths {
		for verb, op := range item {
			m, ok := facadeMethod(t, roxy, path, verb, op)
			if !ok {
				continue
			}
			where := strings.ToUpper(verb) + " " + path
			if body := op.RequestBody.Content["application/json"].Schema; body != nil {
				// Inputs are (receiver, ctx, path params..., params?, body, reqEditors...).
				checked += walkNullable(t, where+" body", body, m.Type.In(m.Type.NumIn()-2), doc.Components.Schemas)
			}
			if resp := op.Responses["200"].Content["application/json"].Schema; resp != nil {
				envelope := m.Type.Out(0).Elem()
				json200, ok := envelope.FieldByName("JSON200")
				if !ok {
					t.Errorf("%s: %s has no JSON200 field", where, envelope)
					continue
				}
				checked += walkNullable(t, where+" 200", resp, json200.Type, doc.Components.Schemas)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no nullable field found on any request or response; the spec shape changed or the walk is broken")
	}
	t.Logf("%d nullable fields checked", checked)
}

type schemaNode struct {
	Type       json.RawMessage            `json:"type"`
	Ref        string                     `json:"$ref"`
	Properties map[string]json.RawMessage `json:"properties"`
	Items      json.RawMessage            `json:"items"`
}

func (n schemaNode) nullable() bool {
	var types []string
	if json.Unmarshal(n.Type, &types) != nil {
		return false
	}
	for _, s := range types {
		if s == "null" {
			return true
		}
	}
	return false
}

// walkNullable descends a schema and the Go type generated for it in step, through
// $ref, arrays and nested objects, and returns how many nullable fields it verified.
// Unions (oneOf, anyOf, allOf) and maps are not walked: the generator represents
// them with its own wrapper types.
func walkNullable(t *testing.T, where string, raw json.RawMessage, goType reflect.Type, schemas map[string]json.RawMessage) int {
	t.Helper()
	var n schemaNode
	if json.Unmarshal(raw, &n) != nil {
		return 0
	}
	if n.Ref != "" {
		name := strings.TrimPrefix(n.Ref, "#/components/schemas/")
		target, ok := schemas[name]
		if !ok {
			t.Errorf("%s: %s does not resolve", where, n.Ref)
			return 0
		}
		return walkNullable(t, name, target, goType, schemas)
	}
	for goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}
	if n.Items != nil && goType.Kind() == reflect.Slice {
		return walkNullable(t, where+"[]", n.Items, goType.Elem(), schemas)
	}
	if goType.Kind() != reflect.Struct || len(n.Properties) == 0 {
		return 0
	}
	checked := 0
	for prop, sub := range n.Properties {
		field, ok := fieldByJSONName(goType, prop)
		if !ok {
			t.Errorf("%s: property %q has no field on %s", where, prop, goType)
			continue
		}
		var child schemaNode
		_ = json.Unmarshal(sub, &child)
		if child.nullable() {
			checked++
			if field.Type.Kind() != reflect.Ptr {
				t.Errorf("%s.%s is nullable in the spec but %s in Go, not a pointer", where, prop, field.Type)
			}
		}
		checked += walkNullable(t, where+"."+prop, sub, field.Type, schemas)
	}
	return checked
}

func fieldByJSONName(st reflect.Type, name string) (reflect.StructField, bool) {
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		if tag := strings.Split(f.Tag.Get("json"), ",")[0]; tag == name {
			return f, true
		}
	}
	return reflect.StructField{}, false
}

// TestLive hits production. It is skipped unless an API key is present, so CI without
// the secret stays green.
func TestLive(t *testing.T) {
	key := os.Getenv("ROXY_API_KEY")
	if key == "" {
		t.Skip("set ROXY_API_KEY to run the live test")
	}
	roxy, err := roxyapi.NewRoxy(key)
	if err != nil {
		t.Fatalf("NewRoxy: %v", err)
	}
	resp, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListZodiacSigns: %v", err)
	}
	if resp.StatusCode() != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		t.Fatal("JSON200 is nil")
	}
}
