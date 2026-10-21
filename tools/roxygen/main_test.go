package main

import (
	"strings"
	"testing"
)

func errSchema(extra map[string]any, required ...any) map[string]any {
	props := map[string]any{
		"error": map[string]any{"type": "string", "example": "x"},
		"code":  map[string]any{"type": "string", "enum": []any{"not_found"}},
	}
	for k, v := range extra {
		props[k] = v
	}
	return map[string]any{"type": "object", "required": required, "properties": props}
}

func specWith(schemas ...map[string]any) map[string]any {
	responses := map[string]any{}
	for i, s := range schemas {
		responses[string(rune('4'))+"0"+string(rune('0'+i))] = map[string]any{
			"content": map[string]any{"application/json": map[string]any{"schema": s}},
		}
	}
	return map[string]any{"paths": map[string]any{"/x": map[string]any{"get": map[string]any{"responses": responses}}}}
}

func TestErrorResponseIsUnionOfSpecSchemas(t *testing.T) {
	suggestions := map[string]any{"type": "array", "description": "closest endpoints", "items": map[string]any{"type": "string"}}
	spec := specWith(
		errSchema(map[string]any{"doc_url": map[string]any{"type": "string", "description": "fix link"}}, "error", "code", "doc_url"),
		errSchema(map[string]any{"suggestions": suggestions}, "error"),
	)
	normalizeErrors(spec)
	er := spec["components"].(map[string]any)["schemas"].(map[string]any)["ErrorResponse"].(map[string]any)
	props := er["properties"].(map[string]any)
	for _, name := range []string{"error", "code", "doc_url", "suggestions"} {
		if props[name] == nil {
			t.Errorf("ErrorResponse lacks %q", name)
		}
	}
	if d := props["suggestions"].(map[string]any)["description"]; d != "closest endpoints" {
		t.Errorf("suggestions description = %v", d)
	}
	if _, pinned := props["code"].(map[string]any)["enum"]; pinned {
		t.Error("a per-operation enum leaked into the shared schema")
	}
	if got := strings.Join(stringList(er["required"]), ","); got != "code,error" {
		t.Errorf("required = %q, want code,error (doc_url is not required everywhere)", got)
	}
}

func TestMergeSchemaFailsOnTypeConflict(t *testing.T) {
	a := errSchema(map[string]any{"extra": map[string]any{"type": "string"}})
	b := errSchema(map[string]any{"extra": map[string]any{"type": "integer"}})
	merged, err := mergeSchema(nil, a, nil, "a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mergeSchema(merged, b, nil, "b"); err == nil || !strings.Contains(err.Error(), "extra") {
		t.Errorf("err = %v, want a conflict naming extra", err)
	}
}
