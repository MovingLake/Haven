package jsonutils

import (
	"fmt"
	"log"
	"reflect"
	"sort"

	"github.com/andres-movl/gojsonschema"
)

// ArraySchema returns the schema for an array.
func ArraySchema(payload []any) map[string]any {
	schema := map[string]any{
		"type": "array",
	}
	if len(payload) == 0 {
		return schema
	}
	types := map[string]bool{}
	for _, v := range payload {
		typ := TypeOf(v)
		types[typ] = true
	}
	if len(types) == 1 {
		typ := TypeOf(payload[0])
		if typ == "object" {
			sch := ObjectSchema(payload[0].(map[string]any))
			schema["items"] = sch
			return schema
		}
		if typ == "array" {
			sch := ArraySchema(payload[0].([]any))
			schema["items"] = sch
			return schema
		}
		schema["items"] = map[string]any{
			"type": typ,
		}
		return schema
	}
	// This is a mixed typed array.
	schema["items"] = map[string]any{
		"anyOf": []any{},
	}
	sortedKeys := []string{}
	for k := range types {
		sortedKeys = append(sortedKeys, k)
	}
	// Helps keep the order consistent.
	sort.Strings(sortedKeys)
	for _, k := range sortedKeys {
		if k == "array" || k == "object" {
			panic("can't handle arrays with mixed nested types")
		}
		schema["items"].(map[string]any)["anyOf"] = append(schema["items"].(map[string]any)["anyOf"].([]any), map[string]any{
			"type": k,
		})
	}
	return schema
}

// ObjectSchema returns the schema for an object.
func ObjectSchema(payload map[string]any) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}
	for k, v := range payload {
		typ := TypeOf(v)
		schema["required"] = append(schema["required"].([]string), k)
		if typ == "object" {
			sch := ObjectSchema(v.(map[string]any))
			schema["properties"].(map[string]any)[k] = sch
			continue
		}
		if typ == "array" {
			sch := ArraySchema(v.([]any))
			schema["properties"].(map[string]any)[k] = sch
			continue
		}
		schema["properties"].(map[string]any)[k] = map[string]any{
			"type": typ,
		}
	}
	// Sort the required properties.
	sort.Strings(schema["required"].([]string))
	return schema
}

// CreateSchema creates a new schema from the payload.
func CreateSchema(payload any, resourceName string) map[string]any {
	schema := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://movinglake.com/haven.schema.json",
		"title":                resourceName,
		"additionalProperties": false, // Very important.
	}
	typ := TypeOf(payload)
	genSchema := map[string]any{}
	if typ == "object" {
		genSchema = ObjectSchema(payload.(map[string]any))
	} else if typ == "array" {
		genSchema = ArraySchema(payload.([]any))
	} else {
		genSchema["properties"] = map[string]any{
			"type": typ,
		}
	}
	for k, v := range genSchema {
		schema[k] = v
	}
	return schema
}

func TypeOf(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case int:
		return "number"
	case float64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
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
func ExpandSchema(schema map[string]any, payload any, errors []gojsonschema.ResultError) error {
	for _, e := range errors {
		fmt.Printf("Error. Type: %s, Details: %v\n", e.Type(), e.Details())
		switch e.Type() {
		case "additional_property_not_allowed":
			if err := additionalPropertyNotAllowed(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add additional property to the schema: %w", err)
			}
		case "invalid_type":
			if err := invalidType(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add invalid type to the schema: %w", err)
			}
		case "required":
			if err := required(e, schema); err != nil {
				return fmt.Errorf("failed to add required property to the schema: %w", err)
			}
		case "array_no_additional_items":
			if err := arrayNoAdditionalItems(); err != nil {
				return fmt.Errorf("failed to add additional property to the schema: %w", err)
			}
		case "array_min_items":
			if err := arrayMinItems(e, schema, payload); err != nil {
				return fmt.Errorf("failed to change minItems on the schema: %w", err)
			}
		case "array_max_items":
			if err := arrayMaxItems(e, schema, payload); err != nil {
				return fmt.Errorf("failed to change maxItems on the schema: %w", err)
			}
		case "unique":
			if err := unique(e, schema); err != nil {
				return fmt.Errorf("failed to add unique property to the schema: %w", err)
			}
		case "contains":
			if err := contains(e, schema); err != nil {
				return fmt.Errorf("failed to add contains property to the schema: %w", err)
			}
		case "array_min_properties":
			if err := arrayMinProperties(); err != nil {
				return fmt.Errorf("failed to add minProperties property to the schema: %w", err)
			}
		case "array_max_properties":
			if err := arrayMaxProperties(); err != nil {
				return fmt.Errorf("failed to add maxProperties property to the schema: %w", err)
			}
		case "invalid_property_pattern":
			if err := invalidPropertyPattern(); err != nil {
				return fmt.Errorf("failed to add invalid property pattern to the schema: %w", err)
			}
		case "invalid_property_name":
			if err := invalidPropertyName(); err != nil {
				return fmt.Errorf("failed to add invalid property name to the schema: %w", err)
			}
		case "string_gte":
			if err := stringGte(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add string greater than or equal to property to the schema: %w", err)
			}
		case "string_lte":
			if err := stringLte(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add string less than or equal to property to the schema: %w", err)
			}
		case "pattern":
			if err := pattern(e, schema); err != nil {
				return fmt.Errorf("failed to add pattern property to the schema: %w", err)
			}
		case "multiple_of":
			if err := multipleOf(e, schema); err != nil {
				return fmt.Errorf("failed to add multiple of property to the schema: %w", err)
			}
		case "number_gte":
			if err := numberGte(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add number greater than or equal to property to the schema: %w", err)
			}
		case "number_gt":
			if err := numberGt(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add number greater than property to the schema: %w", err)
			}
		case "number_lte":
			if err := numberLte(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add number less than or equal to property to the schema: %w", err)
			}
		case "number_lt":
			if err := numberLt(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add number less than property to the schema: %w", err)
			}
		case "condition_then":
			if err := conditionThen(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add condition then property to the schema: %w", err)
			}
		case "condition_else":
			if err := conditionElse(e, schema, payload); err != nil {
				return fmt.Errorf("failed to add condition else property to the schema: %w", err)
			}
		default:
			return fmt.Errorf("unknown schema validation error type: %s", e.Type())
		}
	}
	return nil
}

// ApplyPayload applies the payload to the old schema and returns the new schema and an error if any.
// Note that if no new schema is generated, the newSchema is nil.
func ApplyPayload(oldSchema map[string]any, payload any, resourceName string) (map[string]any, error) {
	if len(oldSchema) == 0 {
		log.Printf("[jsonutils] ApplyPayload got an empty old schema")
		return CreateSchema(payload, resourceName), nil
	}
	schema, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(oldSchema))
	if err != nil {
		return nil, fmt.Errorf("failed to create the schema: %w", err)
	}
	result, err := schema.Validate(gojsonschema.NewGoLoader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to validate the schema: %w", err)
	}

	if result.Valid() {
		return nil, nil
	}

	if err := ExpandSchema(oldSchema, payload, result.Errors()); err != nil {
		return nil, fmt.Errorf("failed to expand the schema: %w", err)
	}
	return oldSchema, nil
}

// ValidatePayload validates the payload against the schema.
func ValidatePayload(schema map[string]any, payload any) (*gojsonschema.Result, error) {
	if len(schema) == 0 {
		return nil, fmt.Errorf("schema is empty")
	}
	goSchema, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(schema))
	if err != nil {
		return nil, fmt.Errorf("failed to create the schema: %w", err)
	}
	return goSchema.Validate(gojsonschema.NewGoLoader(payload))
}
