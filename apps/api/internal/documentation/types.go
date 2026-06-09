package docs

type Response struct {
	Modules []Module `json:"modules"`
}

type Module struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Routes      []Route `json:"routes"`
}

type Route struct {
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	Summary      string  `json:"summary"`
	Description  string  `json:"description"`
	Auth         string  `json:"auth"`
	PathParams   []Field `json:"path_params,omitempty"`
	RequestBody  string  `json:"request_body,omitempty"`
	ResponseBody string  `json:"response_body,omitempty"`
	Errors       []Error `json:"errors,omitempty"`
}

type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type Error struct {
	Status      int    `json:"status"`
	Code        string `json:"code"`
	Description string `json:"description"`
}
