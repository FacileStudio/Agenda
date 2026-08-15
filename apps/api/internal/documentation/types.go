// Package docs re-exports the shared API reference types so each module can
// declare its routes without importing tronc directly.
package docs

import "github.com/FacileStudio/tronc/apiref"

// Response is the shared API reference registry type.
//
// Module, Route, Field and Error alias the corresponding tronc apiref types so
// each module can declare its routes without importing tronc directly.
type (
	Response = apiref.Registry
	Module   = apiref.Module
	Route    = apiref.Route
	Field    = apiref.Field
	Error    = apiref.Error
)
