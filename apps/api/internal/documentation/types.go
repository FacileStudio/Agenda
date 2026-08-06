// Package docs re-exports the shared API reference types so each module can
// declare its routes without importing tronc directly.
package docs

import "github.com/FacileStudio/tronc/apiref"

type (
	Response = apiref.Registry
	Module   = apiref.Module
	Route    = apiref.Route
	Field    = apiref.Field
	Error    = apiref.Error
)
