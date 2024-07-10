package jsonutils_test

import (
	"encoding/json"
	"testing"

	"github.com/andres-movl/gojsonschema"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"movinglake.com/haven/handler/jsonutils"
)

var ignoreSlices = cmpopts.IgnoreSliceElements(func(string) bool { return true })

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
							"type": "integer",
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
			if diff := cmp.Diff(c.want, got, ignoreSlices); diff != "" {
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
						"type": "integer",
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
			if diff := cmp.Diff(c.want, got, ignoreSlices); diff != "" {
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
						"type": "integer",
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
			if diff := cmp.Diff(c.want, got, ignoreSlices); diff != "" {
				t.Errorf("CreateSchema(%v) returned diff %v", c.payload, diff)
			}
		})
	}
}

func TestTypeOf(t *testing.T) {
	cases := []struct {
		payload interface{}
		name    string
		want    string
	}{
		{
			payload: nil,
			name:    "nil",
			want:    "null",
		},
		{
			payload: true,
			name:    "boolean",
			want:    "boolean",
		},
		{
			payload: 1,
			name:    "integer",
			want:    "integer",
		},
		{
			payload: 1.0,
			name:    "number",
			want:    "number",
		},
		{
			payload: "string",
			name:    "string",
			want:    "string",
		},
		{
			payload: map[string]any{},
			name:    "object",
			want:    "object",
		},
		{
			payload: []any{},
			name:    "array",
			want:    "array",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := jsonutils.TypeOf(c.payload)
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("TypeOf(%v) returned diff %v", c.payload, diff)
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
		wantErr error
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
			name: "Key not in schema",
			schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []any{},
			},
			payload: map[string]any{
				"key": "value",
			},
			errors: []gojsonschema.ResultError{
				buildError("additional_property_not_allowed", gojsonschema.NewJsonContext("key", nil), nil, "", "", nil),
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
				},
				"required": []any{"key"},
			},
		},
	}
	// TODO: Run the test cases.
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
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
			name: "Required not in schema",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
					"key2": map[string]any{
						"type": "integer",
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
						"type": "integer",
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
						"type": []any{"string", "integer"},
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
								{"type": "integer"},
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
						"type":       "integer",
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
						"type": "integer",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Maximum Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":    "integer",
						"maximum": 2,
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
						"type":    "integer",
						"maximum": 3,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Exclusive Maximum Violated",
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":             "integer",
						"exclusiveMaximum": 2,
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
						"type":             "integer",
						"exclusiveMaximum": 3,
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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := jsonutils.ApplyPayload(c.schema, c.payload, "")
			if err != nil {
				t.Fatalf("ApplyPayload(%v, %v) returned error %v", c.schema, c.payload, err)
			}
			if diff := cmp.Diff(c.want, got, ignoreSlices); diff != "" {
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
						"type": "integer",
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
					"Invalid type. Expected: string, given: integer",
					"Invalid type. Expected: {{.expected}}, given: {{.given}}",
					gojsonschema.ErrorDetails{
						"context":  string("(root).key"),
						"expected": string("string"),
						"field":    string("key"),
						"given":    string("integer"),
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
				if diff := cmp.Diff(e.Type(), got.Errors()[i].Type(), ignoreSlices); diff != "" {
					t.Errorf("ValidatePayload(%v, %v) returned diff %v", c.schema, c.payload, diff)
				}
			}
		})
	}
}