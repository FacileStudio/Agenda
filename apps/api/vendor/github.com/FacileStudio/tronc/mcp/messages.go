package mcp

import (
	"strconv"
	"time"
)

// MessageDeclined is the standard text for the model once the person
// declined a sensitive action.
const MessageDeclined = "The user declined. Nothing happened."

// Waiting is the standard text for the model when a sensitive action waits
// on the confirm page: nothing ran, hand over confirm_url, and do not ask
// again.
func (c *Confirmations) Waiting() string {
	return waiting(c.cfg.PendingTTL)
}

// waiting is the Waiting text for an action that stays open for ttl.
func waiting(ttl time.Duration) string {
	return "Nothing happened yet: your client cannot ask the user to confirm. Give the user confirm_url and " +
		"ask them to confirm there. Do not call this tool again for the same target; the request stays open for " +
		lifetime(ttl) + "."
}

// lifetime names ttl in whole days, or in hours below a day.
func lifetime(ttl time.Duration) string {
	if days := int(ttl / (24 * time.Hour)); days > 0 {
		return plural(days, "day")
	}
	return plural(max(int(ttl/time.Hour), 1), "hour")
}

// plural is n followed by unit, with an s unless n is 1.
func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return strconv.Itoa(n) + " " + unit + "s"
}
