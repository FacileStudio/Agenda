package bulle

import (
	"encoding/json"

	"github.com/FacileStudio/tronc/errors"
)

type errorFields struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	UpgradeURL string `json:"upgrade_url"`
	Key        string `json:"key"`
}

// envelope reads an error body in tronc's {"error":{...}} envelope or bare,
// with the fields at the top level. A body that is neither gives no fields.
func envelope(raw []byte) errorFields {
	var wrapped struct {
		Error json.RawMessage `json:"error"`
	}
	var fields errorFields
	if json.Unmarshal(raw, &wrapped) == nil && len(wrapped.Error) > 0 && json.Unmarshal(wrapped.Error, &fields) == nil {
		return fields
	}
	fields = errorFields{}
	if json.Unmarshal(wrapped.Error, &fields.Message) == nil {
		return fields
	}
	if json.Unmarshal(raw, &fields) != nil {
		return errorFields{}
	}
	return fields
}

func planRequired(raw []byte) *PlanRequired {
	fields := envelope(raw)
	plan := &PlanRequired{Message: fields.Message, Key: fields.Key}
	if errors.SafeUpgradeURL(fields.UpgradeURL) {
		plan.UpgradeURL = fields.UpgradeURL
	}
	return plan
}
