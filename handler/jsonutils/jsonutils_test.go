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
		payload jsonutils.JArray
		name    string
		want    jsonutils.JObject
	}{
		{
			payload: jsonutils.JArray{"key"},
			name:    "Single string",
			want: jsonutils.JObject{
				"type": "array",
				"items": jsonutils.JObject{
					"type": "string",
				},
			},
		},
		{
			payload: jsonutils.JArray{"key", 2, 3.0, true, nil},
			name:    "Mixed Array",
			want: jsonutils.JObject{
				"type": "array",
				"items": jsonutils.JObject{
					"anyOf": jsonutils.JArray{
						jsonutils.JObject{
							"type": "boolean",
						},
						jsonutils.JObject{
							"type": "integer",
						},
						jsonutils.JObject{
							"type": "null",
						},
						jsonutils.JObject{
							"type": "number",
						},
						jsonutils.JObject{
							"type": "string",
						},
					},
				},
			},
		},
		{
			payload: jsonutils.JArray{},
			name:    "Empty Array",
			want: jsonutils.JObject{
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
		payload jsonutils.JObject
		name    string
		want    jsonutils.JObject
	}{
		{
			payload: jsonutils.JObject{
				"key":  "value",
				"key2": 2,
				"key3": 3.0,
				"key4": true,
				"key5": nil,
				"key6": jsonutils.JObject{
					"key": "value",
				},
				"key7": jsonutils.JArray{"key"},
			},
			name: "Full Object",
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
					"key2": jsonutils.JObject{
						"type": "integer",
					},
					"key3": jsonutils.JObject{
						"type": "number",
					},
					"key4": jsonutils.JObject{
						"type": "boolean",
					},
					"key5": jsonutils.JObject{
						"type": "null",
					},
					"key6": jsonutils.JObject{
						"type":     "object",
						"required": []string{"key"},
						"properties": jsonutils.JObject{
							"key": jsonutils.JObject{
								"type": "string",
							},
						},
					},
					"key7": jsonutils.JObject{
						"type": "array",
						"items": jsonutils.JObject{
							"type": "string",
						},
					},
				},
				"required": []string{"key5", "key", "key2", "key3", "key4", "key6", "key7"},
			},
		},
		{
			payload: jsonutils.JObject{},
			name:    "Empty Object",
			want: jsonutils.JObject{
				"type":       "object",
				"required":   []string{},
				"properties": jsonutils.JObject{},
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
		payload jsonutils.J
		name    string
		want    jsonutils.JObject
	}{
		{
			payload: jsonutils.JObject{
				"key":  "value",
				"key2": 2,
				"key3": 3.0,
				"key4": true,
				"key5": nil,
				"key6": jsonutils.JObject{
					"key": "value",
				},
				"key7": jsonutils.JArray{"key"},
			},
			name: "Full Object",
			want: jsonutils.JObject{
				"$schema":              "https://json-schema.org/draft/2020-12/schema",
				"$id":                  "https://movinglake.com/haven.schema.json",
				"title":                "",
				"type":                 "object",
				"additionalProperties": false,
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
					"key2": jsonutils.JObject{
						"type": "integer",
					},
					"key3": jsonutils.JObject{
						"type": "number",
					},
					"key4": jsonutils.JObject{
						"type": "boolean",
					},
					"key5": jsonutils.JObject{
						"type": "null",
					},
					"key6": jsonutils.JObject{
						"type":     "object",
						"required": []string{"key"},
						"properties": jsonutils.JObject{
							"key": jsonutils.JObject{
								"type": "string",
							},
						},
					},
					"key7": jsonutils.JObject{
						"type": "array",
						"items": jsonutils.JObject{
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
			got := jsonutils.CreateSchema(c.payload)
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
			payload: jsonutils.JObject{},
			name:    "object",
			want:    "object",
		},
		{
			payload: jsonutils.JArray{},
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
		schema  jsonutils.JObject
		payload jsonutils.J
		errors  []gojsonschema.ResultError
		want    jsonutils.JObject
		wantErr error
	}{
		{
			name: "No validation errors",
			schema: jsonutils.JObject{
				"type": "object",
			},
			payload: jsonutils.JObject{
				"key": "value",
			},
			errors: []gojsonschema.ResultError{},
			want: jsonutils.JObject{
				"type": "object",
			},
		},
		{
			name: "Key not in schema",
			schema: jsonutils.JObject{
				"type":       "object",
				"properties": jsonutils.JObject{},
				"required":   []string{},
			},
			payload: jsonutils.JObject{
				"key": "value",
			},
			errors: []gojsonschema.ResultError{
				buildError("additional_property_not_allowed", gojsonschema.NewJsonContext("key", nil), nil, "", "", nil),
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
				},
				"required": []string{"key"},
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
		schema  jsonutils.JObject
		payload jsonutils.J
		want    jsonutils.JObject
	}{
		{
			name: "No changes",
			schema: jsonutils.JObject{
				"type": "object",
			},
			payload: jsonutils.JObject{},
		},
		{
			name: "Required not in schema",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
					"key2": jsonutils.JObject{
						"type": "integer",
					},
				},
				"required":             []string{"key"},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key2": 1,
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
					"key2": jsonutils.JObject{
						"type": "integer",
					},
				},
				"required":             []string{"key2"},
				"additionalProperties": false,
			},
		},
		{
			name: "Type Changed",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
				},
				"required":             []string{"key"},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": 1,
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": []string{"string", "integer"},
					},
				},
				"additionalProperties": false,
				"required":             []string{"key"},
			},
		},
		{
			name: "MaxItems Exceeded",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":     "array",
						"maxItems": 1,
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": jsonutils.JArray{"value", "value2"},
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":     "array",
						"maxItems": 2,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "MinItems Exceeded",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":     "array",
						"minItems": 2,
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": jsonutils.JArray{"value"},
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":     "array",
						"minItems": 1,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Unique Items Violated",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":        "array",
						"uniqueItems": true,
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": jsonutils.JArray{"value", "value"},
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "array",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Contains Violated",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":     "array",
						"contains": jsonutils.JObject{"type": "string"},
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": jsonutils.JArray{1, 2},
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "array",
						"items": jsonutils.JObject{
							"anyOf": []jsonutils.JObject{
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
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":      "string",
						"minLength": 10,
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": "value",
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":      "string",
						"minLength": 5,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "String lte Violated",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":      "string",
						"maxLength": 1,
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": "value",
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":      "string",
						"maxLength": 5,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Pattern Not Matching",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":    "string",
						"pattern": "ask.*",
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": "answer",
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Multiple Of Violated",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":       "integer",
						"multipleOf": 2,
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": 3,
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "integer",
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Maximum Violated",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":    "integer",
						"maximum": 2,
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": 3,
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":    "integer",
						"maximum": 3,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Exclusive Maximum Violated",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":             "integer",
						"exclusiveMaximum": 2,
					},
				},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": 3,
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type":             "integer",
						"exclusiveMaximum": 3,
					},
				},
				"additionalProperties": false,
			},
		},
		{
			name: "Key Added",
			schema: jsonutils.JObject{
				"type":                 "object",
				"properties":           jsonutils.JObject{},
				"required":             []string{},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
				"key": "value",
			},
			want: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
				},
				"additionalProperties": false,
				"required":             []string{"key"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := jsonutils.ApplyPayload(c.schema, c.payload)
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
		schema  jsonutils.JObject
		payload jsonutils.J
		want    []gojsonschema.ResultError
	}{
		{
			name: "No changes",
			schema: jsonutils.JObject{
				"type": "object",
			},
			payload: jsonutils.JObject{},
		},
		{
			name: "Required not in schema",
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
					"key2": jsonutils.JObject{
						"type": "integer",
					},
				},
				"required":             []string{"key"},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
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
			schema: jsonutils.JObject{
				"type": "object",
				"properties": jsonutils.JObject{
					"key": jsonutils.JObject{
						"type": "string",
					},
				},
				"required":             []string{"key"},
				"additionalProperties": false,
			},
			payload: jsonutils.JObject{
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
