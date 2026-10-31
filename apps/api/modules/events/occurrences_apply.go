package events

import (
	"strings"
	"time"

	"github.com/FacileStudio/tronc/errors"

	"github.com/emersion/go-ical"
)

func (c *seriesCtx) applyRequest(comp *ical.Component, req *OccurrenceRequest) error {
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return errors.Invalid("title cannot be empty")
	}
	if err := c.applyTimes(comp, req); err != nil {
		return err
	}
	setOptionalText(comp, ical.PropSummary, req.Title)
	setOptionalText(comp, ical.PropDescription, req.Description)
	setOptionalText(comp, ical.PropLocation, req.Location)
	if req.Status != nil {
		comp.Props.SetText(ical.PropStatus, strings.ToUpper(*req.Status))
	}
	return nil
}

func setOptionalText(comp *ical.Component, name string, value *string) {
	switch {
	case value == nil:
	case *value == "":
		comp.Props.Del(name)
	default:
		comp.Props.SetText(name, *value)
	}
}

func (c *seriesCtx) applyTimes(comp *ical.Component, req *OccurrenceRequest) error {
	if req.StartAt == nil && req.EndAt == nil {
		return nil
	}
	start, _ := comp.Props.DateTime(ical.PropDateTimeStart, c.loc)
	end, _ := comp.Props.DateTime(ical.PropDateTimeEnd, c.loc)
	duration := end.Sub(start)
	if req.StartAt != nil {
		start = *req.StartAt
		end = addWallClock(start, duration)
	}
	if req.EndAt != nil {
		end = *req.EndAt
	}
	if !end.After(start) {
		return errors.Invalid("end time must be after start time")
	}
	comp.Props.Set(c.timeProp(ical.PropDateTimeStart, start))
	comp.Props.Set(c.timeProp(ical.PropDateTimeEnd, end))
	return nil
}

func (c *seriesCtx) newOverride() *ical.Component {
	comp := ical.NewComponent(ical.CompEvent)
	comp.Props.Set(c.master.Props.Get(ical.PropUID))
	comp.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())
	comp.Props.Set(c.timeProp(ical.PropRecurrenceID, c.at))
	comp.Props.Set(c.timeProp(ical.PropDateTimeStart, c.at))
	comp.Props.Set(c.timeProp(ical.PropDateTimeEnd, addWallClock(c.at, c.evt.EndAt.Sub(c.evt.StartAt))))
	for _, name := range []string{ical.PropSummary, ical.PropDescription, ical.PropLocation} {
		if p := c.master.Props.Get(name); p != nil {
			comp.Props.Set(p)
		}
	}
	return comp
}
