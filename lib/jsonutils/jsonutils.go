package jsonutils

import (
	"fmt"
	"reflect"

	"github.com/xeipuuv/gojsonschema"
)

type JObject map[string]interface{}
type JArray []interface{}
type J interface{}

// ArraySchema returns the schema for an array.
func ArraySchema(payload JArray) JObject {
	schema := JObject{
		"type": "array",
	}
	// TODO: Support complicated arrays like arrays of objects with different schemas.
	return schema
}

// ObjectSchema returns the schema for an object.
func ObjectSchema(payload JObject) JObject {
	schema := JObject{
		"type":       "object",
		"properties": JObject{},
		"required":   []string{},
	}
	for k, v := range payload {
		typ := TypeOf(v)
		schema["required"] = append(schema["required"].([]string), k)
		if typ == "object" {
			sch := ObjectSchema(v.(JObject))
			schema["properties"].(JObject)[k] = sch
			continue
		}
		if typ == "array" {
			sch := ArraySchema(v.(JArray))
			schema["properties"].(JObject)[k] = sch
			continue
		}
		schema["properties"].(JObject)[k] = JObject{
			"type": typ,
		}
	}
	return schema
}

// CreateSchema creates a new schema from the payload.
func CreateSchema(payload J) JObject {
	schema := JObject{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     "https://movinglake.com/haven.schema.json",
		"title":   "",
	}
	typ := TypeOf(payload)
	genSchema := JObject{}
	if typ == "object" {
		genSchema = ObjectSchema(payload.(JObject))
	} else if typ == "array" {
		genSchema = ArraySchema(payload.(JArray))
	} else {
		genSchema["properties"] = JObject{
			"type": typ,
		}
	}
	for k, v := range genSchema {
		schema[k] = v
	}
	return schema
}

func TypeOf(v interface{}) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64:
		return "number"
	case int:
		return "integer"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Slice:
		return "array"
	case reflect.Map:
		return "object"
	default:
		return "default"
	}
}

// ExpandSchema expands the old schema with the payload.
func ExpandSchema(oldSchema JObject, payload J, errors []gojsonschema.ResultError) error {
	for _, e := range errors {
		fmt.Println(e)
	}
	return nil
}

// ApplyPayload applies the payload to the old schema and returns the new schema and an error if any.
func ApplyPayload(oldSchema JObject, payload J) (newSchema JObject, err error) {
	if len(oldSchema) == 0 {
		return CreateSchema(payload), nil
	}
	result, err := gojsonschema.Validate(gojsonschema.NewGoLoader(payload), gojsonschema.NewGoLoader(oldSchema))
	if err != nil {
		return nil, err
	}

	if result.Valid() {
		return nil, nil
	}

	if err := ExpandSchema(oldSchema, payload, result.Errors()); err != nil {
		return nil, err
	}
	return nil, nil
}
