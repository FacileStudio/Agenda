package apiref

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// DefaultVersion is the API version reported when a Config leaves it empty.
const DefaultVersion = "1.0.0"

// OpenAPI converts a Config's registry into an OpenAPI 3.1 document.
//
// It declares the bearerAuth security scheme it references, emits path and query
// parameters, generates component schemas for reflected types, and attaches request
// and response bodies where the registry declares them.
func OpenAPI(cfg Config) map[string]any {
	cfg = cfg.withDefaults()

	paths := map[string]any{}
	tags := make([]any, 0, len(cfg.Registry.Modules))
	builder := newSchemaBuilder()

	for _, module := range cfg.Registry.Modules {
		tags = append(tags, map[string]any{
			"name":        module.Name,
			"description": module.Description,
		})
		for _, route := range module.Routes {
			operation := map[string]any{
				"tags":        []any{module.Name},
				"summary":     route.Summary,
				"operationId": operationID(route.Method, route.Path),
				"responses":   builder.responses(route),
			}
			if route.Description != "" {
				operation["description"] = route.Description
			}
			if route.Auth != "" {
				operation["security"] = []any{map[string]any{"bearerAuth": []any{}}}
			}

			var parameters []any
			if params := pathParameters(route.PathParams); len(params) > 0 {
				parameters = append(parameters, params...)
			}
			if query := queryParameters(route.QueryParams); len(query) > 0 {
				parameters = append(parameters, query...)
			}
			if len(parameters) > 0 {
				operation["parameters"] = parameters
			}

			if route.RequestBody != nil && route.RequestBody != "" {
				if schema := builder.resolveSchema(route.RequestBody); schema != nil {
					operation["requestBody"] = map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{"schema": schema},
						},
					}
				}
			}

			item, ok := paths[route.Path].(map[string]any)
			if !ok {
				item = map[string]any{}
				paths[route.Path] = item
			}
			item[strings.ToLower(route.Method)] = operation
		}
	}

	components := map[string]any{
		"securitySchemes": map[string]any{
			"bearerAuth": map[string]any{"type": "http", "scheme": "bearer"},
		},
	}
	if len(builder.schemas) > 0 {
		components["schemas"] = builder.schemas
	}

	document := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       cfg.Title,
			"version":     cfg.Version,
			"description": cfg.Description,
		},
		"tags":       tags,
		"paths":      paths,
		"components": components,
	}

	if len(cfg.Servers) > 0 {
		servers := make([]any, 0, len(cfg.Servers))
		for _, url := range cfg.Servers {
			servers = append(servers, map[string]any{"url": url})
		}
		document["servers"] = servers
	}

	return document
}

func operationID(method, path string) string {
	cleaned := strings.NewReplacer("/", "_", "{", "", "}", "").Replace(path)
	return strings.ToLower(method) + strings.TrimRight("_"+strings.Trim(cleaned, "_"), "_")
}

func pathParameters(fields []Field) []any {
	params := make([]any, 0, len(fields))
	for _, field := range fields {
		params = append(params, map[string]any{
			"name":        field.Name,
			"in":          "path",
			"required":    true,
			"description": field.Description,
			"schema":      map[string]any{"type": openapiType(field.Type)},
		})
	}
	return params
}

func queryParameters(fields []Field) []any {
	params := make([]any, 0, len(fields))
	for _, field := range fields {
		params = append(params, map[string]any{
			"name":        field.Name,
			"in":          "query",
			"description": field.Description,
			"schema":      map[string]any{"type": openapiType(field.Type)},
		})
	}
	return params
}

type schemaBuilder struct {
	schemas map[string]any
}

func newSchemaBuilder() *schemaBuilder {
	return &schemaBuilder{schemas: map[string]any{}}
}

func (b *schemaBuilder) responses(route Route) map[string]any {
	status := route.Status
	if status == 0 {
		status = http.StatusOK
	}
	statusStr := fmt.Sprintf("%d", status)

	description := "Success"
	if route.ResponseBody != nil {
		if s, ok := route.ResponseBody.(string); ok && s != "" {
			description = s
		}
	}

	out := map[string]any{}
	if status == http.StatusNoContent {
		out[statusStr] = map[string]any{"description": "No Content"}
	} else {
		success := map[string]any{"description": description}
		if route.ResponseBody != nil && route.ResponseBody != "" {
			if schema := b.resolveSchema(route.ResponseBody); schema != nil {
				success["content"] = map[string]any{
					"application/json": map[string]any{"schema": schema},
				}
			}
		}
		out[statusStr] = success
	}

	for _, failure := range route.Errors {
		out[fmt.Sprintf("%d", failure.Status)] = map[string]any{
			"description": strings.TrimSpace(failure.Code + " " + failure.Description),
		}
	}
	return out
}

func (b *schemaBuilder) resolveSchema(target any) map[string]any {
	if target == nil {
		return nil
	}

	if s, ok := target.(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		if inner, found := strings.CutPrefix(s, "[]"); found {
			return map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "object", "description": inner},
			}
		}
		return map[string]any{"type": "object", "description": s}
	}

	var typ reflect.Type
	if t, ok := target.(reflect.Type); ok {
		typ = t
	} else {
		typ = reflect.TypeOf(target)
	}

	return b.reflectType(typ)
}

func (b *schemaBuilder) reflectType(typ reflect.Type) map[string]any {
	if typ == nil {
		return nil
	}

	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ.PkgPath() == "time" && typ.Name() == "Time" {
		return map[string]any{"type": "string", "format": "date-time"}
	}

	switch typ.Kind() {
	case reflect.Struct:
		name := typ.Name()
		if name != "" {
			if _, exists := b.schemas[name]; !exists {
				b.schemas[name] = map[string]any{"type": "object"}
				b.schemas[name] = b.reflectStruct(typ)
			}
			return map[string]any{"$ref": "#/components/schemas/" + name}
		}
		return b.reflectStruct(typ)

	case reflect.Slice, reflect.Array:
		elem := typ.Elem()
		return map[string]any{
			"type":  "array",
			"items": b.reflectType(elem),
		}

	case reflect.Map:
		return map[string]any{"type": "object"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}

	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}

	case reflect.Bool:
		return map[string]any{"type": "boolean"}

	case reflect.String:
		return map[string]any{"type": "string"}

	default:
		return map[string]any{"type": "string"}
	}
}

func (b *schemaBuilder) reflectStruct(typ reflect.Type) map[string]any {
	properties := map[string]any{}
	var required []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		omitempty := false
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					omitempty = true
				}
			}
		} else {
			fieldName = strings.ToLower(fieldName[:1]) + fieldName[1:]
		}

		validateTag := field.Tag.Get("validate")
		isRequired := strings.Contains(validateTag, "required") ||
			(!omitempty && field.Type.Kind() != reflect.Pointer && field.Type.Kind() != reflect.Slice && field.Type.Kind() != reflect.Map)

		fieldSchema := b.reflectType(field.Type)
		if doc := field.Tag.Get("doc"); doc != "" {
			fieldSchema["description"] = doc
		}

		properties[fieldName] = fieldSchema
		if isRequired {
			required = append(required, fieldName)
		}
	}

	res := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		res["required"] = required
	}
	return res
}

func openapiType(kind string) string {
	switch kind {
	case "int", "integer", "int64":
		return "integer"
	case "bool", "boolean":
		return "boolean"
	case "number", "float", "float64":
		return "number"
	default:
		return "string"
	}
}
