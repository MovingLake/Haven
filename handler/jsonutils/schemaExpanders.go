package jsonutils

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/andres-movl/gojsonschema"
)

func valueAt(ctx *gojsonschema.JsonContext, payload any) any {
	if ctx.Head() == "(root)" || ctx.Tail() == nil {
		return payload
	}
	if isArray(ctx) {
		return valueAt(ctx.Tail(), payload.(map[string]any)[arrayName(ctx)])
	}
	return valueAt(ctx.Tail(), payload.(map[string]any)[ctx.Head()])
}

func schemaValueAt(ctx *gojsonschema.JsonContext, schema map[string]any) (any, error) {
	// Need to traverse the list and then access the schema in reverse order.
	// This is because the schema is nested in the reverse order of the path.
	path := []string{}
	for ctx.Head() != "(root)" {
		path = append(path, ctx.Head())
		ctx = ctx.Tail()
	}
	if len(path) == 0 {
		return schema, nil
	}
	curr := schema
	for i := len(path) - 1; i >= 0; i-- {
		if _, err := strconv.Atoi(path[i]); err == nil {
			// An array, move into the items.
			tmp, ok := curr["items"]
			if !ok {
				return nil, fmt.Errorf("\"items\" not found in %v", curr)
			}
			curr, ok = tmp.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("error casting \"items\" in %v", curr)
			}
			continue
		}
		tmp, ok := curr["properties"]
		if !ok {
			return nil, fmt.Errorf("\"properties\" not found in %v", curr)
		}
		curr, ok = tmp.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("error casting \"properties\" in %v", curr)
		}
		tmp, ok = curr[path[i]]
		if !ok {
			return nil, fmt.Errorf("%v not found in %v", path[i], curr)
		}
		curr, ok = tmp.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("error casting %v in %v", path[i], curr)
		}
	}
	return curr, nil
}

func additionalPropertyNotAllowed(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	prop, ok := e.Details()["property"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	typ := TypeOf(payload.(map[string]any)[propName])

	propTmp, ok := schema["properties"]
	if !ok {
		schema["properties"] = map[string]any{}
		propTmp = schema["properties"]
	}
	props, ok := propTmp.(map[string]any)
	if !ok {
		return fmt.Errorf("properties is not a map but %T", propTmp)
	}
	// Add the property to the schema.
	if typ == "object" {
		props[propName] = ObjectSchema(payload.(map[string]any)[propName].(map[string]any))
		return nil
	}
	if typ == "array" {
		props[propName] = ArraySchema(payload.(map[string]any)[propName].([]any))
		return nil
	}

	props[propName] = map[string]any{
		"type": typ,
	}
	return nil
}

// isArray checks if the context is an array.
func isArray(ctx *gojsonschema.JsonContext) bool {
	if _, err := strconv.Atoi(ctx.Head()); err == nil {
		return true
	}
	return false
}

// arrayName returns the name of the array.
func arrayName(ctx *gojsonschema.JsonContext) string {
	return ctx.Tail().Head()
}

func invalidType(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	if isArray(e.Context()) {
		// Invalid type can be because an item in the array is of a different type.
		// Add the type to the items schema.
		propName := arrayName(e.Context())
		prevType := schema["properties"].(map[string]any)[propName].(map[string]any)["items"]
		if prevType == nil {
			prevType = e.Details()["expected"].(string)
		} else if reflect.TypeOf(prevType).Kind() == reflect.String {
			prevType = prevType.(string)
		}
		typ := e.Details()["given"].(string)
		schema["properties"].(map[string]any)[propName].(map[string]any)["items"] = map[string]any{
			"anyOf": []map[string]any{
				{
					"type": prevType,
				},
				{
					"type": typ,
				},
			},
		}
		return nil
	}
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	typ := e.Details()["given"].(string)
	// This is a change of type. Make the type an array if not already and add the type.
	prevType := e.Details()["expected"]
	if reflect.TypeOf(prevType).Kind() == reflect.String {
		prevType = []any{prevType}
	}
	prevType = append(prevType.([]any), typ)
	if typ == "object" {
		schema["properties"].(map[string]any)[propName] = ObjectSchema(payload.(map[string]any)[propName].(map[string]any))
		schema["properties"].(map[string]any)[propName].(map[string]any)["type"] = prevType
		return nil
	}
	if typ == "array" {
		schema["properties"].(map[string]any)[propName] = ArraySchema(payload.(map[string]any)[propName].([]any))
		schema["properties"].(map[string]any)[propName].(map[string]any)["type"] = prevType
		return nil
	}
	schema["properties"].(map[string]any)[propName] = map[string]any{
		"type": prevType,
	}
	return nil

}

func required(e gojsonschema.ResultError, schema map[string]any) error {
	prop, ok := e.Details()["property"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	indexOfProp := -1
	obj, err := schemaValueAt(e.Context(), schema)
	if err != nil {
		return fmt.Errorf("failed to get required properties from the schema: %w", err)
	}
	tmp, ok := obj.(map[string]any)["required"]
	if !ok {
		return fmt.Errorf("required properties not found in the schema")
	}
	casted, ok := tmp.([]any)
	if !ok {
		return fmt.Errorf("required properties is not an array but %T", tmp)
	}
	for i, p := range casted {
		if p == prop {
			indexOfProp = i
			break
		}
	}
	if indexOfProp == -1 {
		// Probably removed by a previous error.
		return nil
	}
	obj.(map[string]any)["required"] = append(casted[:indexOfProp], casted[indexOfProp+1:]...)
	return nil
}

func arrayNoAdditionalItems() error {
	// TODO Implement this. Not sure how to trigger this error.
	return fmt.Errorf("not implemented")
}

func arrayMaxItems(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	schema["properties"].(map[string]any)[propName].(map[string]any)["maxItems"] = len(payload.(map[string]any)[propName].([]any))
	return nil
}

func arrayMinItems(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	schema["properties"].(map[string]any)[propName].(map[string]any)["minItems"] = len(payload.(map[string]any)[propName].([]any))
	return nil
}

func unique(e gojsonschema.ResultError, schema map[string]any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	delete(schema["properties"].(map[string]any)[propName].(map[string]any), "uniqueItems")
	return nil
}

func contains(e gojsonschema.ResultError, schema map[string]any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	delete(schema["properties"].(map[string]any)[propName].(map[string]any), "contains")
	return nil
}

func arrayMinProperties() error {
	return fmt.Errorf("not implemented")
}

func arrayMaxProperties() error {
	return fmt.Errorf("not implemented")
}

func invalidPropertyPattern() error {
	return fmt.Errorf("not implemented")
}

func invalidPropertyName() error {
	return fmt.Errorf("not implemented")
}

func stringGte(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	schema["properties"].(map[string]any)[propName].(map[string]any)["minLength"] = len(payload.(map[string]any)[propName].(string))
	return nil
}

func stringLte(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	schema["properties"].(map[string]any)[propName].(map[string]any)["maxLength"] = len(payload.(map[string]any)[propName].(string))
	return nil
}

func pattern(e gojsonschema.ResultError, schema map[string]any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	// TODO Expand the pattern to include the new string.
	delete(schema["properties"].(map[string]any)[propName].(map[string]any), "pattern")
	return nil
}

func multipleOf(e gojsonschema.ResultError, schema map[string]any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	// TODO: Maybe add the greatest common divisor of the multipleOf values.
	delete(schema["properties"].(map[string]any)[propName].(map[string]any), "multipleOf")
	return nil
}

func numberGte(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	var p any
	p, ok = payload.(map[string]any)[propName].(float64)
	if !ok {
		p = payload.(map[string]any)[propName].(int)
	}
	schema["properties"].(map[string]any)[propName].(map[string]any)["minimum"] = p
	return nil
}

func numberLte(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	var p any
	p, ok = payload.(map[string]any)[propName].(float64)
	if !ok {
		p = payload.(map[string]any)[propName].(int)
	}
	schema["properties"].(map[string]any)[propName].(map[string]any)["maximum"] = p
	return nil
}

func numberGt(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	var p any
	p, ok = payload.(map[string]any)[propName].(float64)
	if !ok {
		p = payload.(map[string]any)[propName].(int)
	}
	schema["properties"].(map[string]any)[propName].(map[string]any)["exclusiveMinimum"] = p
	return nil
}

func numberLt(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	var p any
	p, ok = payload.(map[string]any)[propName].(float64)
	if !ok {
		p = payload.(map[string]any)[propName].(int)
	}
	schema["properties"].(map[string]any)[propName].(map[string]any)["exclusiveMaximum"] = p
	return nil
}

func conditionThen(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	// TODO Implement this. Not sure how to trigger this error.
	return fmt.Errorf("not implemented")
}

func conditionElse(e gojsonschema.ResultError, schema map[string]any, payload any) error {
	// TODO Implement this. Not sure how to trigger this error.
	return fmt.Errorf("not implemented")
}
