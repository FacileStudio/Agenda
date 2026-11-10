package forfait

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/FacileStudio/porte/directory"
)

// exportVersion is the only document version Import accepts.
const exportVersion = 1

// maxExport is the largest document Import reads. Export holds a subset of a
// snapshot's bytes, but re-encoding can grow a string up to three times (an
// invalid UTF-8 byte becomes U+FFFD), so four snapshot caps keep every
// document Export can produce from a snapshot Client accepted importable.
const maxExport = 4 * directory.MaxSnapshotBytes

// ErrAlreadyApplied is what Import answers once this Store applied a
// snapshot: the state in memory is newer than anything saved before it, so
// the Store keeps it and the call changes nothing.
var ErrAlreadyApplied = errors.New("forfait: import after a snapshot was applied")

// exported is the persisted shape of a Store's applied state.
type exported struct {
	Version              int                                   `json:"version"`
	SnapshotVersion      int64                                 `json:"snapshot_version"`
	EntitlementsRevision int64                                 `json:"entitlements_revision"`
	Spaces               map[string]map[string]json.RawMessage `json:"spaces"`
	People               map[string]map[string]json.RawMessage `json:"people"`
}

// Export is the applied state as a versioned JSON document, so an app does
// not fail open again after a restart. Save it, as one row or one file, after
// every Apply, and hand it to Import at boot before the first pull. It
// carries the snapshot Version and EntitlementsRevision; a Store still
// failing open exports revision 0 and imports as failing open.
func (s *Store) Export() ([]byte, error) {
	s.mu.RLock()
	document := exported{Version: exportVersion, SnapshotVersion: s.version, EntitlementsRevision: s.revision}
	if s.revision > 0 {
		document.Spaces = s.encodeScopes(s.spaces)
		document.People = s.encodeScopes(s.people)
	}
	s.mu.RUnlock()
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(document); err != nil {
		return nil, fmt.Errorf("forfait: encode the export: %w", err)
	}
	return out.Bytes(), nil
}

// Import restores a document Export produced. Empty input (nothing saved
// yet, on a first boot) is a no-op that returns nil, so an app passes
// whatever its storage holds without a special case. It reads at most four
// directory.MaxSnapshotBytes, rejects unknown fields, trailing data, negative
// versions and any document version but the current one, and filters every
// value through the same rules as Apply: undeclared keys, malformed values
// and scopes over 200 keys read as defaults. It refuses to run once this
// Store applied a snapshot, with ErrAlreadyApplied, and on any error the Store is left unchanged,
// still failing open. The restored Version and EntitlementsRevision then
// guard Apply against an older snapshot.
func (s *Store) Import(b []byte) error {
	if len(bytes.TrimSpace(b)) == 0 {
		return nil
	}
	document, err := decodeExport(b)
	if err != nil {
		return err
	}
	spaces, people := s.indexScopes(document.Spaces), s.indexScopes(document.People)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revision > 0 {
		return ErrAlreadyApplied
	}
	if document.EntitlementsRevision == 0 {
		return nil
	}
	s.spaces, s.people = spaces, people
	s.version, s.revision = document.SnapshotVersion, document.EntitlementsRevision
	return nil
}

// decodeExport parses and bounds one document, without touching any Store.
func decodeExport(b []byte) (exported, error) {
	var document exported
	if len(b) > maxExport {
		return document, fmt.Errorf("forfait: import of %d bytes exceeds %d", len(b), maxExport)
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return document, fmt.Errorf("forfait: decode the import: %w", err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return document, errors.New("forfait: trailing data after the import")
	}
	if document.Version != exportVersion {
		return document, fmt.Errorf("forfait: import version %d is not %d", document.Version, exportVersion)
	}
	if document.SnapshotVersion < 0 || document.EntitlementsRevision < 0 {
		return document, errors.New("forfait: import carries a negative version")
	}
	return document, nil
}

// indexScopes filters every persisted scope through index.
func (s *Store) indexScopes(scopes map[string]map[string]json.RawMessage) map[string]map[string]value {
	resolved := make(map[string]map[string]value, len(scopes))
	for id, entitlements := range scopes {
		resolved[id] = index(s.features, entitlements)
	}
	return resolved
}

// encodeScopes writes every scope's values back as the JSON Apply reads.
func (s *Store) encodeScopes(scopes map[string]map[string]value) map[string]map[string]json.RawMessage {
	encoded := make(map[string]map[string]json.RawMessage, len(scopes))
	for id, values := range scopes {
		entitlements := make(map[string]json.RawMessage, len(values))
		for key, v := range values {
			entitlements[key] = encodeValue(s.features[key].Kind, v)
		}
		encoded[id] = entitlements
	}
	return encoded
}

// encodeValue is v as a JSON bool for a Bool and a JSON integer for a Limit.
func encodeValue(kind Kind, v value) json.RawMessage {
	if kind == Bool {
		return json.RawMessage(strconv.FormatBool(v.enabled))
	}
	return json.RawMessage(strconv.FormatInt(v.limit, 10))
}
