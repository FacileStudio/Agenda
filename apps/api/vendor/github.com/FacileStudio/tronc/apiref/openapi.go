package apiref

import (
	"fmt"
	"strings"
)

// DefaultVersion is the API version reported when a Config leaves it empty.
const DefaultVersion = "1.0.0"

// OpenAPI converts a Config's registry into an OpenAPI 3.1 document.
//
// It declares the bearerAuth security scheme it references, emits path
// parameters, and attaches request and response bodies where the registry names
// them. Routes whose Auth is empty are documented as public.
func OpenAPI(cfg Config) map[string]any {
	cfg = cfg.withDefaults()

	paths := map[string]any{}
	tags := make([]any, 0, len(cfg.Registry.Modules))

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
				"responses":   responses(route),
			}
			if route.Description != "" {
				operation["description"] = route.Description
			}
			if route.Auth != "" {
				operation["security"] = []any{map[string]any{"bearerAuth": []any{}}}
			}
			if params := pathParameters(route.PathParams); len(params) > 0 {
				operation["parameters"] = params
			}
			if route.RequestBody != "" {
				operation["requestBody"] = map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": map[string]any{"schema": namedSchema(route.RequestBody)},
					},
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

	document := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       cfg.Title,
			"version":     cfg.Version,
			"description": cfg.Description,
		},
		"tags":  tags,
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{"type": "http", "scheme": "bearer"},
			},
		},
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

func responses(route Route) map[string]any {
	description := route.ResponseBody
	if description == "" {
		description = "Success"
	}
	success := map[string]any{"description": description}
	if route.ResponseBody != "" {
		success["content"] = map[string]any{
			"application/json": map[string]any{"schema": namedSchema(route.ResponseBody)},
		}
	}
	out := map[string]any{"200": success}
	for _, failure := range route.Errors {
		out[fmt.Sprintf("%d", failure.Status)] = map[string]any{
			"description": strings.TrimSpace(failure.Code + " " + failure.Description),
		}
	}
	return out
}

func namedSchema(name string) map[string]any {
	if inner, ok := strings.CutPrefix(name, "[]"); ok {
		return map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "object", "description": inner},
		}
	}
	return map[string]any{"type": "object", "description": name}
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
