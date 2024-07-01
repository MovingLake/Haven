package jsonutils

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/andres-movl/gojsonschema"
)

func additionalPropertyNotAllowed(e gojsonschema.ResultError, schema JObject, payload J) error {
	prop, ok := e.Details()["property"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	typ := TypeOf(payload.(JObject)[propName])
	// Add the property to the schema.
	if typ == "object" {
		schema["properties"].(JObject)[propName] = ObjectSchema(payload.(JObject)[propName].(JObject))
		return nil
	}
	if typ == "array" {
		schema["properties"].(JObject)[propName] = ArraySchema(payload.(JObject)[propName].(JArray))
		return nil
	}
	schema["properties"].(JObject)[propName] = JObject{
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

func invalidType(e gojsonschema.ResultError, schema JObject, payload J) error {
	if isArray(e.Context()) {
		// Invalid type can be because an item in the array is of a different type.
		// Add the type to the items schema.
		propName := arrayName(e.Context())
		prevType := schema["properties"].(JObject)[propName].(JObject)["items"]
		if prevType == nil {
			prevType = e.Details()["expected"].(string)
		} else if reflect.TypeOf(prevType).Kind() == reflect.String {
			prevType = prevType.(string)
		}
		typ := e.Details()["given"].(string)
		schema["properties"].(JObject)[propName].(JObject)["items"] = JObject{
			"anyOf": []JObject{
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
		prevType = []string{prevType.(string)}
	}
	prevType = append(prevType.([]string), typ)
	if typ == "object" {
		schema["properties"].(JObject)[propName] = ObjectSchema(payload.(JObject)[propName].(JObject))
		schema["properties"].(JObject)[propName].(JObject)["type"] = prevType
		return nil
	}
	if typ == "array" {
		schema["properties"].(JObject)[propName] = ArraySchema(payload.(JObject)[propName].(JArray))
		schema["properties"].(JObject)[propName].(JObject)["type"] = prevType
		return nil
	}
	schema["properties"].(JObject)[propName] = JObject{
		"type": prevType,
	}
	return nil

}

func required(e gojsonschema.ResultError, schema JObject) error {
	prop, ok := e.Details()["property"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	// Just remove the required property.
	indexOfProp := -1
	for i, p := range schema["required"].([]string) {
		if p == prop {
			indexOfProp = i
			break
		}
	}
	if indexOfProp == -1 {
		return fmt.Errorf("property not found in the required properties")
	}
	schema["required"] = append(schema["required"].([]string)[:indexOfProp], schema["required"].([]string)[indexOfProp+1:]...)
	return nil
}

func arrayNoAdditionalItems() error {
	// TODO Implement this. Not sure how to trigger this error.
	return fmt.Errorf("not implemented")
}

func arrayMaxItems(e gojsonschema.ResultError, schema JObject, payload J) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	schema["properties"].(JObject)[propName].(JObject)["maxItems"] = len(payload.(JObject)[propName].(JArray))
	return nil
}

func arrayMinItems(e gojsonschema.ResultError, schema JObject, payload J) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	schema["properties"].(JObject)[propName].(JObject)["minItems"] = len(payload.(JObject)[propName].(JArray))
	return nil
}

func unique(e gojsonschema.ResultError, schema JObject) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	delete(schema["properties"].(JObject)[propName].(JObject), "uniqueItems")
	return nil
}

func contains(e gojsonschema.ResultError, schema JObject) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	delete(schema["properties"].(JObject)[propName].(JObject), "contains")
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

func stringGte(e gojsonschema.ResultError, schema JObject, payload J) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	schema["properties"].(JObject)[propName].(JObject)["minLength"] = len(payload.(JObject)[propName].(string))
	return nil
}

func stringLte(e gojsonschema.ResultError, schema JObject, payload J) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	schema["properties"].(JObject)[propName].(JObject)["maxLength"] = len(payload.(JObject)[propName].(string))
	return nil
}

func pattern(e gojsonschema.ResultError, schema JObject) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	// TODO Expand the pattern to include the new string.
	delete(schema["properties"].(JObject)[propName].(JObject), "pattern")
	return nil
}

func multipleOf(e gojsonschema.ResultError, schema JObject) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	// TODO: Maybe add the greatest common divisor of the multipleOf values.
	delete(schema["properties"].(JObject)[propName].(JObject), "multipleOf")
	return nil
}

func numberGte(e gojsonschema.ResultError, schema JObject, payload J) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	var p any
	p, ok = payload.(JObject)[propName].(float64)
	if !ok {
		p = payload.(JObject)[propName].(int)
	}
	schema["properties"].(JObject)[propName].(JObject)["minimum"] = p
	return nil
}

func numberLte(e gojsonschema.ResultError, schema JObject, payload J) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	var p any
	p, ok = payload.(JObject)[propName].(float64)
	if !ok {
		p = payload.(JObject)[propName].(int)
	}
	schema["properties"].(JObject)[propName].(JObject)["maximum"] = p
	return nil
}

func numberGt(e gojsonschema.ResultError, schema JObject, payload J) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	var p any
	p, ok = payload.(JObject)[propName].(float64)
	if !ok {
		p = payload.(JObject)[propName].(int)
	}
	schema["properties"].(JObject)[propName].(JObject)["exclusiveMinimum"] = p
	return nil
}

func numberLt(e gojsonschema.ResultError, schema JObject, payload J) error {
	prop, ok := e.Details()["field"]
	if !ok {
		return fmt.Errorf("property not found in the error details")
	}
	propName, ok := prop.(string)
	if !ok {
		return fmt.Errorf("property is not a string")
	}
	var p any
	p, ok = payload.(JObject)[propName].(float64)
	if !ok {
		p = payload.(JObject)[propName].(int)
	}
	schema["properties"].(JObject)[propName].(JObject)["exclusiveMaximum"] = p
	return nil
}

func conditionThen(e gojsonschema.ResultError, schema JObject, payload J) error {
	// TODO Implement this. Not sure how to trigger this error.
	return fmt.Errorf("not implemented")
}

func conditionElse(e gojsonschema.ResultError, schema JObject, payload J) error {
	// TODO Implement this. Not sure how to trigger this error.
	return fmt.Errorf("not implemented")
}
