package events

import (
	"strings"
	"time"

	"github.com/FacileStudio/tronc/errors"

	"github.com/emersion/go-ical"
)

func cancelOccurrence(c *seriesCtx) error {
	dropOverride(c)
	if !isExcluded(c) {
		c.master.Props.Add(c.timeProp(ical.PropExceptionDates, c.at))
	}
	return nil
}

func restoreOccurrence(c *seriesCtx) error {
	dropOverride(c)
	dropExdate(c)
	return nil
}

func overrideOccurrence(req *OccurrenceRequest) seriesEdit {
	return func(c *seriesCtx) error {
		if isExcluded(c) {
			return errors.Invalid("the occurrence is cancelled; restore it before overriding it")
		}
		comp := dropOverride(c)
		if comp == nil {
			comp = c.newOverride()
		}
		if err := c.applyRequest(comp, req); err != nil {
			return err
		}
		c.cal.Children = append(c.cal.Children, comp)
		return nil
	}
}

func (c *seriesCtx) timeProp(name string, t time.Time) *ical.Prop {
	p := ical.NewProp(name)
	switch {
	case c.evt.IsAllDay:
		p.Params.Set(ical.ParamValue, "DATE")
		p.Value = t.UTC().Format("20060102")
	case c.tzid != "":
		p.Params.Set(ical.PropTimezoneID, c.tzid)
		p.Value = t.In(c.loc).Format("20060102T150405")
	default:
		p.Value = t.UTC().Format("20060102T150405Z")
	}
	return p
}

func (c *seriesCtx) isAt(prop *ical.Prop) bool {
	t, err := prop.DateTime(c.loc)
	return err == nil && t.Unix() == c.at.Unix()
}

func isExcluded(c *seriesCtx) bool {
	for _, t := range exceptionDates(c.master, c.loc) {
		if t.Unix() == c.at.Unix() {
			return true
		}
	}
	return false
}

func dropExdate(c *seriesCtx) {
	props := c.master.Props.Values(ical.PropExceptionDates)
	c.master.Props.Del(ical.PropExceptionDates)
	for _, p := range props {
		var keep []string
		for v := range strings.SplitSeq(p.Value, ",") {
			single := p
			single.Value = v
			if !c.isAt(&single) {
				keep = append(keep, v)
			}
		}
		if len(keep) > 0 {
			p.Value = strings.Join(keep, ",")
			c.master.Props.Add(&p)
		}
	}
}

func dropOverride(c *seriesCtx) *ical.Component {
	var found *ical.Component
	kept := c.cal.Children[:0]
	for _, child := range c.cal.Children {
		if rid := child.Props.Get(ical.PropRecurrenceID); child.Name == ical.CompEvent && rid != nil && c.isAt(rid) {
			found = child
			continue
		}
		kept = append(kept, child)
	}
	c.cal.Children = kept
	return found
}
