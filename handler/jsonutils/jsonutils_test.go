package jsonutils_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/andres-movl/gojsonschema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"movinglake.com/haven/handler/jsonutils"
)

var sortSlices = cmpopts.SortSlices(func(a, b any) bool {
	astr, ok := a.(string)
	bstr, okb := b.(string)
	if !ok || !okb {
		return false
	}
	return astr < bstr
})

func buildError(t string, c *gojsonschema.JsonContext, v any, d, df string, dets gojsonschema.ErrorDetails) gojsonschema.ResultError {
	f := &gojsonschema.ResultErrorFields{}
	f.SetType(t)
	f.SetValue(v)
	f.SetContext(c)
	f.SetDescription(d)
	f.SetDescriptionFormat(df)
	f.SetDetails(dets)
	return f
}

func TestArraySchema(t *testing.T) {
	cases := []struct {
		payload []any
		name    string
		want    map[string]any
	}{
		{
			payload: []any{"key"},
			name:    "Single string",
			want: map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		{
			payload: []any{"key", 2, 3.0, true, nil},
			name:    "Mixed Array",
			want: map[string]any{
				"type": "array",
				"items": map[string]any{
					"anyOf": []any{
						map[string]any{
							"type": "boolean",
						},
						map[string]any{
							"type": "null",
						},
						map[string]any{
							"type": "number",
						},
						map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
		{
			payload: []any{},
			name:    "Empty Array",
			want: map[string]any{
				"type": "array",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := jsonutils.ArraySchema(c.payload)
			if diff := cmp.Diff(c.want, got, sortSlices); diff != "" {
				t.Errorf("ArraySchema(%v) returned diff %v", c.payload, diff)
			}
		})
	}
}

func TestObjectSchema(t *testing.T) {
	cases := []struct {
		payload map[string]any
		name    string
		want    map[string]any
	}{
		{
			payload: map[string]any{
				"key":  "value",
				"key2": 2,
				"key3": 3.0,
				"key4": true,
				"key5": nil,
				"key6": map[string]any{
					"key": "value",
				},
				"key7": []any{"key"},
			},
			name: "Full Object",
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
					"key2": map[string]any{
						"type": "number",
					},
					"key3": map[string]any{
						"type": "number",
					},
					"key4": map[string]any{
						"type": "boolean",
					},
					"key5": map[string]any{
						"type": "null",
					},
					"key6": map[string]any{
						"type":     "object",
						"required": []string{"key"},
						"properties": map[string]any{
							"key": map[string]any{
								"type": "string",
							},
						},
					},
					"key7": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required": []string{"key5", "key", "key2", "key3", "key4", "key6", "key7"},
			},
		},
		{
			payload: map[string]any{},
			name:    "Empty Object",
			want: map[string]any{
				"type":       "object",
				"required":   []string{},
				"properties": map[string]any{},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := jsonutils.ObjectSchema(c.payload)
			if diff := cmp.Diff(c.want, got, sortSlices); diff != "" {
				t.Errorf("ObjectSchema(%v) returned diff %v", c.payload, diff)
			}
		})
	}
}

func TestCreateSchema(t *testing.T) {
	cases := []struct {
		payload any
		name    string
		want    map[string]any
	}{
		{
			payload: map[string]any{
				"key":  "value",
				"key2": 2,
				"key3": 3.0,
				"key4": true,
				"key5": nil,
				"key6": map[string]any{
					"key": "value",
				},
				"key7": []any{"key"},
			},
			name: "Full Object",
			want: map[string]any{
				"$schema":              "https://json-schema.org/draft/2020-12/schema",
				"$id":                  "https://movinglake.com/haven.schema.json",
				"title":                "Full Object",
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
					"key2": map[string]any{
						"type": "number",
					},
					"key3": map[string]any{
						"type": "number",
					},
					"key4": map[string]any{
						"type": "boolean",
					},
					"key5": map[string]any{
						"type": "null",
					},
					"key6": map[string]any{
						"type":     "object",
						"required": []string{"key"},
						"properties": map[string]any{
							"key": map[string]any{
								"type": "string",
							},
						},
					},
					"key7": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required": []string{"key5", "key", "key2", "key3", "key4", "key6", "key7"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := jsonutils.CreateSchema(c.payload, c.name)
			if diff := cmp.Diff(c.want, got, sortSlices); diff != "" {
				t.Errorf("CreateSchema(%v) returned diff %v", c.payload, diff)
			}
		})
	}
}

func TestTypeOf(t *testing.T) {
	cases := []struct {
		payload string
		name    string
		want    string
	}{
		{
			payload: "null",
			name:    "nil",
			want:    "null",
		},
		{
			payload: "true",
			name:    "boolean",
			want:    "boolean",
		},
		{
			payload: "1",
			name:    "number",
			want:    "number",
		},
		{
			payload: "1.0",
			name:    "number",
			want:    "number",
		},
		{
			payload: "\"string\"",
			name:    "string",
			want:    "string",
		},
		{
			payload: "{}",
			name:    "object",
			want:    "object",
		},
		{
			payload: "[]",
			name:    "array",
			want:    "array",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			str := fmt.Sprintf("{\"a\": %s}", c.payload)
			jstr := map[string]any{}
			err := json.Unmarshal([]byte(str), &jstr)
			if err != nil {
				t.Fatalf("json.Unmarshal(%s) returned error %v", str, err)
			}
			got := jsonutils.TypeOf(jstr["a"])
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("TypeOf(%v) returned diff %v", jstr["a"], diff)
			}
		})
	}
}

func TestExpandSchema(t *testing.T) {
	cases := []struct {
		name    string
		schema  map[string]any
		payload any
		errors  []gojsonschema.ResultError
		want    map[string]any
		wantErr bool
	}{
		{
			name: "No validation errors",
			schema: map[string]any{
				"type": "object",
			},
			payload: map[string]any{
				"key": "value",
			},
			errors: []gojsonschema.ResultError{},
			want: map[string]any{
				"type": "object",
			},
		},
		{
			name: "Additional property fails",
			errors: []gojsonschema.ResultError{
				buildError("additional_property_not_allowed", gojsonschema.NewJsonContext("key", nil), nil, "", "", gojsonschema.ErrorDetails{}),
			},
			wantErr: true,
		},
		{
			name: "Invalid Type property fails",
			errors: []gojsonschema.ResultError{
				buildError("invalid_type", gojsonschema.NewJsonContext("key", nil), nil, "", "", gojsonschema.ErrorDetails{}),
			},
			wantErr: true,
		},
		{
			name: "Required property fails",
			errors: []gojsonschema.ResultError{
				buildError("required", gojsonschema.NewJsonContext("key", nil), nil, "", "", gojsonschema.ErrorDetails{}),
			},
			wantErr: true,
		},
		{
			name: "Array Max Items Fails",
			errors: []gojsonschema.ResultError{
				buildError("array_no_additional_items", gojsonschema.NewJsonContext("key", nil), nil, "", "", gojsonschema.ErrorDetails{}),
			},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := jsonutils.ExpandSchema(c.schema, c.payload, c.errors)
			if c.wantErr && err == nil {
				t.Fatalf("ExpandSchema(%v, %v, %v) returned nil, expected error", c.schema, c.payload, c.errors)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("ExpandSchema(%v, %v, %v) returned error %v", c.schema, c.payload, c.errors, err)
			}
			if c.wantErr {
				return
			}

			if diff := cmp.Diff(c.want, c.schema, sortSlices); diff != "" {
				t.Errorf("ExpandSchema(%v, %v, %v) returned diff %v", c.schema, c.payload, c.errors, diff)
			}

		})
	}
}

func TestApplyPayload(t *testing.T) {
	cases := []struct {
		name    string
		schema  map[string]any
		payload any
		want    map[string]any
	}{
		{
			name: "No changes",
			schema: map[string]any{
				"type": "object",
			},
			payload: map[string]any{},
		},
		{
			name: "Empty schema changes",
			schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
			},
			payload: map[string]any{
				"name": "John Doe",
				"age":  30,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"age": map[string]any{
						"type": "number",
					},
					"name": map[string]any{
						"type": "string",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Required not in schema",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
					"key2": map[string]any{
						"type": "number",
					},
				},
				"required":             []any{"key"},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key2": 1,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
					"key2": map[string]any{
						"type": "number",
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
		{
			name: "Required not in schema nested",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"k1": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"k2": map[string]any{
								"type": "string",
							},
						},
						"required": []any{"k2"},
					},
				},
				"required":             []any{"k1"},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"k1": map[string]any{},
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"k1": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"k2": map[string]any{
								"type": "string",
							},
						},
						"required": []any{},
					},
				},
				"required":             []any{"k1"},
				"additionalProperties": false,
			},
		},
		{
			name: "Type Changed",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
				},
				"required":             []any{"key"},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": 1,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": []any{"number", "string"},
					},
				},
				"additionalProperties": false,
				"required":             []any{"key"},
			},
		},
		{
			name: "MaxItems Exceeded",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":     "array",
						"maxItems": 1,
					},
				},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": []any{"value", "value2"},
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":     "array",
						"maxItems": 2,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "MinItems Exceeded",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":     "array",
						"minItems": 2,
					},
				},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": []any{"value"},
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":     "array",
						"minItems": 1,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Unique Items Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":        "array",
						"uniqueItems": true,
					},
				},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": []any{"value", "value"},
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "array",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Contains Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":     "array",
						"contains": map[string]any{"type": "string"},
					},
				},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": []any{1, 2},
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "array",
						"items": map[string]any{
							"anyOf": []map[string]any{
								{"type": "string"},
								{"type": "number"},
							},
						},
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "String gte Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":      "string",
						"minLength": 10,
					},
				},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": "value",
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":      "string",
						"minLength": 5,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "String lte Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":      "string",
						"maxLength": 1,
					},
				},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": "value",
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":      "string",
						"maxLength": 5,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Pattern Not Matching",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":    "string",
						"pattern": "ask.*",
					},
				},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": "answer",
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Multiple Of Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":       "number",
						"multipleOf": 2,
					},
				},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": 3,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "number",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Key Added",
			schema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": "value",
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
				},
				"additionalProperties": false,
				"required":             []any{},
			},
		},
		{
			name: "Number Gt Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type":    "number",
						"minimum": 5,
					},
					"b": map[string]any{
						"type":    "number",
						"minimum": 4.5,
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"a": 4,
				"b": 4.4,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type":    "number",
						"minimum": 4,
					},
					"b": map[string]any{
						"type":    "number",
						"minimum": 4.4,
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
		{
			name: "Number Gte Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type":             "number",
						"exclusiveMinimum": 5,
					},
					"b": map[string]any{
						"type":             "number",
						"exclusiveMinimum": 4.899,
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"a": 5,
				"b": 4.899,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type":             "number",
						"exclusiveMinimum": 4,
					},
					"b": map[string]any{
						"type":             "number",
						"exclusiveMinimum": 4.8989,
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
		{
			name: "Number Lt Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type":    "number",
						"maximum": 5,
					},
					"b": map[string]any{
						"type":    "number",
						"maximum": 4.5,
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"a": 6,
				"b": 4.6,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type":    "number",
						"maximum": 6,
					},
					"b": map[string]any{
						"type":    "number",
						"maximum": 4.6,
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
		{
			name: "Number Lte Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type":             "number",
						"exclusiveMaximum": 5,
					},
					"b": map[string]any{
						"type":             "number",
						"exclusiveMaximum": 4.899,
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"a": 5,
				"b": 4.899,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type":             "number",
						"exclusiveMaximum": 6,
					},
					"b": map[string]any{
						"type":             "number",
						"exclusiveMaximum": 4.8991,
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
		{
			name: "Type changes string to null",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": "string",
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"a": nil,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": []any{"null", "string"},
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
		{
			name: "Type changes null to string",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": "null",
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"a": "b",
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": []any{"null", "string"},
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
		{
			name: "Type changes null to number",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": "null",
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"a": 1233,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": []any{"null", "number"},
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
		{
			name: "Nested type changes number to null",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"b": map[string]any{
								"type": "number",
							},
						},
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"a": map[string]any{"b": nil},
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"b": map[string]any{
								"type": []any{"null", "number"},
							},
						},
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
		{
			name: "Type changes null to number",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": "null",
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"a": 1233,
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{
						"type": []any{"null", "number"},
					},
				},
				"required":             []any{},
				"additionalProperties": false,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := jsonutils.ApplyPayload(c.schema, c.payload, "")
			if err != nil {
				t.Fatalf("ApplyPayload(%v, %v) returned error %v", c.schema, c.payload, err)
			}
			if diff := cmp.Diff(c.want, got, sortSlices); diff != "" {
				t.Errorf("ApplyPayload(%v, %v) returned diff %v", c.schema, c.payload, diff)
			}
		})
	}
}

func TestValidatePayload(t *testing.T) {
	cases := []struct {
		name    string
		schema  map[string]any
		payload any
		want    []gojsonschema.ResultError
	}{
		{
			name: "No changes",
			schema: map[string]any{
				"type": "object",
			},
			payload: map[string]any{},
		},
		{
			name: "Required not in schema",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
					"key2": map[string]any{
						"type": "number",
					},
				},
				"required":             []any{"key"},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key2": 1,
			},
			want: []gojsonschema.ResultError{
				buildError(
					"required",
					gojsonschema.NewJsonContext("(root)", nil),
					map[string]any{"key2": json.Number("1")},
					"key is required",
					"{{.property}} is required",
					gojsonschema.ErrorDetails{
						"context":  string("(root)"),
						"field":    string("(root)"),
						"property": string("key"),
					},
				),
			},
		},
		{
			name: "Type Changed",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
				},
				"required":             []any{"key"},
				"additionalProperties": false,
			},
			payload: map[string]any{
				"key": 1,
			},
			want: []gojsonschema.ResultError{
				buildError(
					"invalid_type",
					gojsonschema.NewJsonContext("key", gojsonschema.NewJsonContext("(root)", nil)),
					map[string]any{"key2": json.Number("1")},
					"Invalid type. Expected: string, given: number",
					"Invalid type. Expected: {{.expected}}, given: {{.given}}",
					gojsonschema.ErrorDetails{
						"context":  string("(root).key"),
						"expected": string("string"),
						"field":    string("key"),
						"given":    string("number"),
					},
				),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := jsonutils.ValidatePayload(c.schema, c.payload)
			if err != nil {
				t.Fatalf("ValidatePayload(%v, %v) returned error %v", c.schema, c.payload, err)
			}
			for i, e := range c.want {
				if diff := cmp.Diff(e.Type(), got.Errors()[i].Type(), sortSlices); diff != "" {
					t.Errorf("ValidatePayload(%v, %v) returned diff %v", c.schema, c.payload, diff)
				}
			}
		})
	}
}
