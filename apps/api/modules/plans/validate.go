package plans

import (
	stderrors "errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/FacileStudio/tronc/errors"
)

// plain returns s as one line of plain text: control and invisible format
// characters (zero width, bidi overrides) are dropped or turned into a
// space, runs of spaces collapse, and the ends are trimmed. The text is
// never parsed afterwards, whatever it looks like.
func plain(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsSpace(r) || unicode.IsControl(r):
			b.WriteRune(' ')
		case unicode.Is(unicode.Cf, r):
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// text cleans s and refuses it when it is longer than limit characters.
func text(field, s string, limit int, required bool) (string, error) {
	out := plain(s)
	if required && out == "" {
		return "", errors.Invalid(field + " is required")
	}
	if utf8.RuneCountInString(out) > limit {
		return "", errors.Invalid(fmt.Sprintf("%s is at most %d characters", field, limit))
	}
	return out, nil
}

// link validates a task link: an http or https URL with a host and no
// credentials. An empty link is allowed.
func link(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if len(s) > MaxTaskURL || strings.ContainsFunc(s, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) {
		return "", errors.Invalid(fmt.Sprintf("task_url is at most %d characters and holds no spaces", MaxTaskURL))
	}
	parsed, err := url.Parse(s)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil {
		return "", errors.Invalid("task_url must be an http or https link")
	}
	return parsed.String(), nil
}

// zoneName checks an IANA zone, defaulting to UTC.
func zoneName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "UTC", nil
	}
	if _, err := time.LoadLocation(name); err != nil || strings.EqualFold(name, "local") {
		return "", errors.Invalid(fmt.Sprintf("timezone %q is not an IANA name such as Europe/Paris", name))
	}
	return name, nil
}

// clean validates and normalizes a proposal against now: sizes, plain text,
// links, times inside the next four weeks and no two blocks overlapping each
// other. Calendars and busy time are checked by the service, which holds the
// database.
func clean(in ProposeInput, now time.Time) (ProposeInput, error) {
	var err error
	out := ProposeInput{}
	if out.Title, err = text("title", in.Title, MaxPlanTitle, true); err != nil {
		return out, err
	}
	if out.Timezone, err = zoneName(in.Timezone); err != nil {
		return out, err
	}
	if len(in.Warnings) > MaxWarnings {
		return out, errors.Invalid(fmt.Sprintf("a plan holds at most %d warnings", MaxWarnings))
	}
	out.Warnings = []string{}
	for _, w := range in.Warnings {
		if w, err = text("a warning", w, MaxWarning, false); err != nil {
			return out, err
		}
		if w != "" {
			out.Warnings = append(out.Warnings, w)
		}
	}
	if len(in.Blocks) == 0 || len(in.Blocks) > MaxBlocks {
		return out, errors.Invalid(fmt.Sprintf("a plan holds 1 to %d blocks", MaxBlocks))
	}
	for i, b := range in.Blocks {
		cleaned, err := cleanBlock(b, now)
		if err != nil {
			return out, atBlock(i, err)
		}
		out.Blocks = append(out.Blocks, cleaned)
	}
	return out, overlapping(out.Blocks)
}

// atBlock prefixes a block's refusal with its place in the plan.
func atBlock(i int, err error) error {
	var known *errors.Error
	if stderrors.As(err, &known) {
		return errors.Invalid(fmt.Sprintf("block %d: %s", i+1, known.Message))
	}
	return err
}

func cleanBlock(b BlockInput, now time.Time) (BlockInput, error) {
	var err error
	if b.Title, err = text("title", b.Title, MaxBlockTitle, true); err != nil {
		return b, err
	}
	if b.Reason, err = text("reason", b.Reason, MaxReason, false); err != nil {
		return b, err
	}
	if b.TaskRef, err = text("task_ref", b.TaskRef, MaxTaskRef, false); err != nil {
		return b, err
	}
	if b.TaskURL, err = link(b.TaskURL); err != nil {
		return b, err
	}
	if b.CalendarID <= 0 {
		return b, errors.Invalid("calendar_id is required")
	}
	b.StartAt, b.EndAt = b.StartAt.UTC(), b.EndAt.UTC()
	return b, window(b, now)
}

// window checks the length of a block and that it lies in the next four weeks.
func window(b BlockInput, now time.Time) error {
	switch length := b.EndAt.Sub(b.StartAt); {
	case b.StartAt.IsZero() || b.EndAt.IsZero():
		return errors.Invalid("start and end are required")
	case length < MinBlock || length > MaxBlock:
		return errors.Invalid(fmt.Sprintf("a block lasts %d minutes to %d hours", int(MinBlock.Minutes()), int(MaxBlock.Hours())))
	case b.StartAt.Before(now):
		return errors.Invalid("the block starts in the past")
	case b.EndAt.After(now.Add(Horizon)):
		return errors.Invalid("the block ends after the next 4 weeks")
	}
	return nil
}

// overlapping refuses blocks that overlap each other, since nobody does two
// things at once.
func overlapping(blocks []BlockInput) error {
	order := make([]int, len(blocks))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int { return blocks[a].StartAt.Compare(blocks[b].StartAt) })
	for i := 1; i < len(order); i++ {
		prev, next := order[i-1], order[i]
		if blocks[next].StartAt.Before(blocks[prev].EndAt) {
			return errors.Invalid(fmt.Sprintf("blocks %d and %d overlap", min(prev, next)+1, max(prev, next)+1))
		}
	}
	return nil
}
