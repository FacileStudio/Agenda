package docs

import (
	"fmt"
	"strings"
)

type OpenAPISpec struct {
	OpenAPI string                 `json:"openapi"`
	Info    OpenAPIInfo            `json:"info"`
	Paths   map[string]interface{} `json:"paths"`
}

type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

func ToOpenAPI(resp Response) OpenAPISpec {
	spec := OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: OpenAPIInfo{
			Title:       "Courrier API",
			Description: "Courrier email client API",
			Version:     "1.0.0",
		},
		Paths: make(map[string]interface{}),
	}

	for _, mod := range resp.Modules {
		for _, route := range mod.Routes {
			path := route.Path
			method := strings.ToLower(route.Method)

			operation := map[string]interface{}{
				"summary":     route.Summary,
				"description": route.Description,
				"tags":        []string{mod.Name},
				"operationId": method + "_" + strings.ReplaceAll(strings.Trim(path, "/"), "/", "_"),
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Success",
					},
				},
			}

			if route.Auth != "" {
				operation["security"] = []map[string][]string{
					{"bearerAuth": {}},
				}
			}

			errResponses := operation["responses"].(map[string]interface{})
			for _, e := range route.Errors {
				statusStr := fmt.Sprintf("%d", e.Status)
				errResponses[statusStr] = map[string]interface{}{
					"description": e.Description,
				}
			}

			if _, ok := spec.Paths[path]; !ok {
				spec.Paths[path] = make(map[string]interface{})
			}
			spec.Paths[path].(map[string]interface{})[method] = operation
		}
	}

	return spec
}
