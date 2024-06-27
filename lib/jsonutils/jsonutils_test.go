package jsonutils_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"movinglake.com/haven/lib/jsonutils"
)

var ignoreSlices = cmpopts.IgnoreSliceElements(func(string) bool { return true })

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
			},
		},
		{
			payload: jsonutils.JArray{"key", 2, 3.0, true, nil},
			name:    "Mixed Array",
			want: jsonutils.JObject{
				"type": "array",
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
	// Write your test code here
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
	// Write your test code here
}

func TestApplyPayload(t *testing.T) {
	// Write your test code here
}
