package apiref

import (
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

// qualifier matches the package path in front of a type argument's name.
var qualifier = regexp.MustCompile(`[\w./-]*\.`)

// unsafeName matches what an OpenAPI component name may not contain.
var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// schemaName is the component name of a named type. An instance of a
// generic type, which reflect names with its arguments' package paths,
// becomes the type name and its arguments' names joined by underscores:
// PendingList[example.com/app.Item] is PendingList_Item.
func schemaName(typ reflect.Type) string {
	base, args, generic := strings.Cut(typ.Name(), "[")
	if !generic {
		return base
	}
	args = qualifier.ReplaceAllString(strings.TrimSuffix(args, "]"), "")
	return base + "_" + strings.Trim(unsafeName.ReplaceAllString(args, "_"), "_")
}

// embeddedStruct reports whether field is an embedded struct, by value,
// whose fields encoding/json promotes into the outer object.
func embeddedStruct(field reflect.StructField) (reflect.Type, bool) {
	untagged := strings.Split(field.Tag.Get("json"), ",")[0] == ""
	return field.Type, field.Anonymous && untagged && field.Type.Kind() == reflect.Struct
}

// promote adds the properties of an embedded struct's schema that the outer
// struct does not declare itself, as encoding/json lets the shallower field
// win, and returns required with the promoted required names.
func promote(properties map[string]any, required []string, inner map[string]any) []string {
	innerRequired, _ := inner["required"].([]string)
	innerProperties, _ := inner["properties"].(map[string]any)
	for _, name := range slices.Sorted(maps.Keys(innerProperties)) {
		if _, taken := properties[name]; taken {
			continue
		}
		properties[name] = innerProperties[name]
		if slices.Contains(innerRequired, name) {
			required = append(required, name)
		}
	}
	return required
}

// property is what one struct field contributes to its object's schema.
type property struct {
	name     string
	schema   map[string]any
	required bool
}

// reflectField is the property one exported field contributes. ok is false
// for a field JSON skips.
func (b *schemaBuilder) reflectField(field reflect.StructField) (property, bool) {
	jsonTag := field.Tag.Get("json")
	if !field.IsExported() || jsonTag == "-" {
		return property{}, false
	}

	fieldName := field.Name
	omitempty := false
	if jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] != "" {
			fieldName = parts[0]
		}
		omitempty = slices.Contains(parts[1:], "omitempty")
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
	return property{name: fieldName, schema: fieldSchema, required: isRequired}, true
}
