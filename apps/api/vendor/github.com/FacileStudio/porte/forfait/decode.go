package forfait

import (
	"bytes"
	"encoding/json"
	"regexp"
)

// maxKeys is the most entitlement keys one scope may carry. A scope with more
// is treated as malformed and every feature in it falls back to its default.
const maxKeys = 200

// keyPattern is the only shape a feature key may take.
var keyPattern = regexp.MustCompile(`^[a-z0-9_]+\.[a-z0-9_]+$`)

// value is one resolved entitlement: enabled for a Bool, limit for a Limit.
type value struct {
	enabled bool
	limit   int64
}

// decode reads raw as a value of kind, reporting false for anything but a
// JSON bool for a Bool or a JSON integer of at least -1 for a Limit.
func decode(kind Kind, raw json.RawMessage) (value, bool) {
	trimmed := bytes.TrimSpace(raw)
	switch {
	case kind == Bool && (string(trimmed) == "true" || string(trimmed) == "false"):
		return value{enabled: string(trimmed) == "true"}, true
	case kind == Limit && startsAsInteger(trimmed):
		var limit int64
		if json.Unmarshal(trimmed, &limit) != nil || limit < -1 {
			return value{}, false
		}
		return value{limit: limit}, true
	}
	return value{}, false
}

// index keeps the well-formed values of the declared features in one scope.
// A scope over maxKeys yields nothing, so it reads as its defaults.
func index(features map[string]Feature, entitlements map[string]json.RawMessage) map[string]value {
	resolved := map[string]value{}
	if len(entitlements) > maxKeys {
		return resolved
	}
	for key, raw := range entitlements {
		feature, declared := features[key]
		if !declared {
			continue
		}
		if v, ok := decode(feature.Kind, raw); ok {
			resolved[key] = v
		}
	}
	return resolved
}

// startsAsInteger rejects every JSON token an int64 could silently decode
// from, such as null, before json.Unmarshal sees it.
func startsAsInteger(raw []byte) bool {
	return len(raw) > 0 && (raw[0] == '-' || (raw[0] >= '0' && raw[0] <= '9'))
}
