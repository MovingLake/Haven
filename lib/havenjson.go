package lib

type hjson map[string]interface{}

func objectSchema(payload hjson) hjson {
	schema := hjson{
		"type":       "object",
		"properties": hjson{},
		"required":   []string{},
	}
	for k, v := range payload {
		typ := typeOf(v)
		schema["required"] = append(schema["required"].([]string), k)
		if typ == "object" {
			sch := objectSchema(v.(hjson), k)
			schema["properties"].(hjson)[k] = sch
			continue
		}
		if typ == "array" {
			sch := arraySchema(v.(hjson), k)
			schema["properties"].(hjson)[k] = sch
			continue
		}
		schema["properties"].(hjson)[k] = hjson{
			"type": typ,
		}
	}
	return schema
}

func createSchema(payload hjson, name string) hjson {
	schema := hjson{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     "https://movinglake.com/haven.schema.json",
		"title":   name,
	}
	sch := objectSchema(v.(hjson), k)
	for k, v := range sch {
		schema[k] = v
	}
	return schema
}

func typeOf(v interface{}) string {
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
	default:
		return "unknown"
	}
}

// computeSchemaDiff computes the diff between the old schema and the new payload.
func computeSchemaDiff(oldSchema hjson, payload hjson) (diffs hjson, newSchema hjson, err error) {
	createSchema(payload)
	return nil, nil, nil
}

func createSchema(payload hjson, name string) {
	schema := hjson{
		"$schema":    "https://json-schema.org/draft/2020-12/schema",
		"$id":        "https://movinglake.com/haven.schema.json",
		"title":      name,
		"type":       "object",
		"properties": hjson{},
	}

	for k, v := range payload {
		schema["properties"].(hjson)[k] = hjson{
			"type": typeOf(v),
		}
	}
}

// computeSchemaDiff computes the diff between the old schema and the new payload.
func computeSchemaDiff(oldSchema hjson, payload hjson) (diffs hjson, newSchema hjson, err error) {
	createSchema(payload)
	return nil, nil, nil
}
