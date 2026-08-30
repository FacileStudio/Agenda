package apiref

// Registry is a hand-written inventory of an API's routes, grouped by module.
// It is the single source the reference page and the OpenAPI document are both
// built from, so the two can never disagree with each other.
type Registry struct {
	Modules []Module `json:"modules"`
}

// Module is one functional area of the API — auth, projects, secrets.
type Module struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Routes      []Route `json:"routes"`
}

// Route describes one method-and-path pair. Path is written the way chi writes
// it, with brace parameters such as /projects/{id}, which is also how OpenAPI
// spells them.
type Route struct {
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	Summary      string  `json:"summary"`
	Description  string  `json:"description"`
	Auth         string  `json:"auth"`
	PathParams   []Field `json:"path_params,omitempty"`
	QueryParams  []Field `json:"query_params,omitempty"`
	RequestBody  any     `json:"request_body,omitempty"`
	ResponseBody any     `json:"response_body,omitempty"`
	Status       int     `json:"status,omitempty"`
	Errors       []Error `json:"errors,omitempty"`
}

// Field is one named value, used for path parameters.
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Error is one non-2xx outcome a route can produce.
type Error struct {
	Status      int    `json:"status"`
	Code        string `json:"code"`
	Description string `json:"description"`
}
