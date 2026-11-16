// Package plans holds the draft schedules a model proposes for a person.
//
// A plan is a list of blocks of time for tasks. Proposing one changes no
// calendar: the person reads it in Agenda and applies all of it, some of it
// or none. Applying creates the events through the events service in one
// transaction. Bulle and every other MCP client only propose.
package plans

import (
	"context"
	stderrors "errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service saves, lists, applies and dismisses plans.
type Service struct {
	db     *gorm.DB
	events *events.Service
	now    func() time.Time
}

// NewService builds the service over the database and the events service.
func NewService(db *gorm.DB, eventService *events.Service) *Service {
	return &Service{db: db, events: eventService, now: time.Now}
}

// Propose validates a proposal against the person's calendars and saves it
// as a draft. It refuses a block on a calendar the person cannot write to,
// one that overlaps an event they already have, and a person holding too
// many open drafts. It changes no calendar.
func (s *Service) Propose(ctx context.Context, userID int64, in ProposeInput) (*PlanResponse, error) {
	now := s.now().UTC()
	in, err := clean(in, now)
	if err != nil {
		return nil, err
	}
	space, err := s.checkCalendars(ctx, userID, in.Blocks)
	if err != nil {
		return nil, err
	}
	if clashes, err := s.clashes(ctx, userID, in.Blocks); err != nil {
		return nil, err
	} else if len(clashes) > 0 {
		return nil, errors.Conflict("Blocks overlapping an existing event: " + joined(clashes) + ". Call agenda_find_free_slots and pick free time")
	}
	plan := schemas.Plan{
		UserID: userID, SpaceID: space, Title: in.Title, Timezone: in.Timezone, Warnings: in.Warnings,
		State: schemas.PlanProposed, ExpiresAt: now.Add(ProposedTTL),
	}
	for i, b := range in.Blocks {
		plan.Blocks = append(plan.Blocks, schemas.PlanBlock{
			Position: i, TaskRef: b.TaskRef, TaskURL: b.TaskURL, Title: b.Title, StartAt: b.StartAt,
			EndAt: b.EndAt, CalendarID: b.CalendarID, Reason: b.Reason,
		})
	}
	if err := s.save(ctx, &plan, now); err != nil {
		return nil, err
	}
	return s.respond(plan), nil
}

func (s *Service) save(ctx context.Context, plan *schemas.Plan, now time.Time) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at <= ?", now).Delete(&schemas.Plan{}).Error; err != nil {
			return err
		}
		var open int64
		err := tx.Model(&schemas.Plan{}).Where("user_id = ? AND state = ?", plan.UserID, schemas.PlanProposed).Count(&open).Error
		if err != nil {
			return err
		}
		if open >= MaxOpenPlans {
			return errors.Failed(fmt.Sprintf("%d plans are already waiting for the user. Ask them to apply or dismiss some first", MaxOpenPlans))
		}
		return tx.Create(plan).Error
	})
	var known *errors.Error
	if stderrors.As(err, &known) {
		return err
	}
	if err != nil {
		return errors.Internal("failed to save the plan", err)
	}
	return nil
}

// checkCalendars refuses a block on a calendar the person cannot write to
// and returns the space every block shares, if there is one.
func (s *Service) checkCalendars(ctx context.Context, userID int64, blocks []BlockInput) (*int64, error) {
	seen := map[int64]*int64{}
	var spaces []*int64
	for i, b := range blocks {
		space, known := seen[b.CalendarID]
		if !known {
			access, found, err := schemas.LoadCalendarAccess(s.db.WithContext(ctx), userID, b.CalendarID)
			if err != nil {
				return nil, errors.Internal("failed to load calendar", err)
			}
			if !found || access.Role == "" {
				return nil, errors.NotFound(fmt.Sprintf("block %d: calendar %d not found", i+1, b.CalendarID))
			}
			if access.Role == "reader" {
				return nil, errors.Forbidden(fmt.Sprintf("block %d: calendar %d is read-only", i+1, b.CalendarID))
			}
			space = access.SpaceID
			seen[b.CalendarID] = space
		}
		spaces = append(spaces, space)
	}
	return commonSpace(spaces), nil
}

// commonSpace is the space id every entry holds, or nil when one differs or
// is not in a space.
func commonSpace(spaces []*int64) *int64 {
	if len(spaces) == 0 || spaces[0] == nil {
		return nil
	}
	for _, s := range spaces[1:] {
		if s == nil || *s != *spaces[0] {
			return nil
		}
	}
	id := *spaces[0]
	return &id
}

type interval struct{ start, end time.Time }

// clashes lists the 1-based positions of the blocks that overlap an event on
// any calendar the person can open. All-day and cancelled events do not count
// as busy, like in agenda_find_free_slots.
func (s *Service) clashes(ctx context.Context, userID int64, blocks []BlockInput) ([]int, error) {
	from, to := blocks[0].StartAt, blocks[0].EndAt
	for _, b := range blocks {
		from, to = earliest(from, b.StartAt), latest(to, b.EndAt)
	}
	busy, err := s.busy(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	var out []int
	for i, b := range blocks {
		if slices.ContainsFunc(busy, func(e interval) bool { return e.start.Before(b.EndAt) && e.end.After(b.StartAt) }) {
			out = append(out, i+1)
		}
	}
	return out, nil
}

func (s *Service) busy(ctx context.Context, userID int64, from, to time.Time) ([]interval, error) {
	calendars, err := schemas.AccessibleCalendars(s.db.WithContext(ctx), userID)
	if err != nil {
		return nil, errors.Internal("failed to list calendars", err)
	}
	var busy []interval
	for _, c := range calendars {
		list, err := s.events.ListEvents(ctx, userID, c.ID, &from, &to, time.UTC)
		if err != nil {
			return nil, err
		}
		for _, e := range list {
			if !e.IsAllDay && e.Status != "cancelled" {
				busy = append(busy, interval{start: e.StartAt, end: e.EndAt})
			}
		}
	}
	return busy, nil
}

func earliest(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

func latest(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func joined(numbers []int) string {
	names := make([]string, len(numbers))
	for i, n := range numbers {
		names[i] = fmt.Sprint(n)
	}
	return strings.Join(names, ", ")
}

// List returns the person's plans that have not expired, newest first.
// state, when set, keeps one state.
func (s *Service) List(ctx context.Context, userID int64, state string, limit int) ([]PlanResponse, error) {
	if state != "" && state != schemas.PlanProposed && state != schemas.PlanApplied && state != schemas.PlanDismissed {
		return nil, errors.Invalid("state is proposed, applied or dismissed")
	}
	if limit <= 0 {
		limit = defaultListSize
	}
	q := s.db.WithContext(ctx).Preload("Blocks", func(db *gorm.DB) *gorm.DB { return db.Order("position") }).
		Where("user_id = ? AND expires_at > ?", userID, s.now().UTC())
	if state != "" {
		q = q.Where("state = ?", state)
	}
	var rows []schemas.Plan
	if err := q.Order("created_at DESC, id DESC").Limit(min(limit, maxListSize)).Find(&rows).Error; err != nil {
		return nil, errors.Internal("failed to list plans", err)
	}
	out := make([]PlanResponse, len(rows))
	for i, p := range rows {
		out[i] = *s.respond(p)
	}
	return out, nil
}

// Get returns one of the person's plans. Another person's plan is not found.
func (s *Service) Get(ctx context.Context, userID, planID int64) (*PlanResponse, error) {
	plan, err := s.load(s.db.WithContext(ctx), userID, planID, false)
	if err != nil {
		return nil, err
	}
	return s.respond(*plan), nil
}

func (s *Service) load(db *gorm.DB, userID, planID int64, lock bool) (*schemas.Plan, error) {
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var plan schemas.Plan
	err := db.Preload("Blocks", func(db *gorm.DB) *gorm.DB { return db.Order("position") }).
		Where("id = ? AND user_id = ?", planID, userID).First(&plan).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound("plan not found")
	}
	if err != nil {
		return nil, errors.Internal("failed to load the plan", err)
	}
	return &plan, nil
}

func (s *Service) respond(p schemas.Plan) *PlanResponse {
	out := &PlanResponse{
		ID: p.ID, SpaceID: p.SpaceID, Title: p.Title, Timezone: p.Timezone, State: p.State,
		Expired:  p.State == schemas.PlanProposed && !p.ExpiresAt.After(s.now()),
		Warnings: append([]string{}, p.Warnings...), Blocks: make([]BlockResponse, len(p.Blocks)),
		CreatedAt: p.CreatedAt, ExpiresAt: p.ExpiresAt,
	}
	for i, b := range p.Blocks {
		out.Blocks[i] = BlockResponse{
			ID: b.ID, TaskRef: b.TaskRef, TaskURL: b.TaskURL, Title: b.Title, StartAt: b.StartAt, EndAt: b.EndAt,
			CalendarID: b.CalendarID, Reason: b.Reason, EventID: b.EventID,
		}
	}
	return out
}

// Dismiss closes a proposed plan without touching the calendar. Dismissing a
// dismissed plan changes nothing; an applied plan cannot be dismissed.
func (s *Service) Dismiss(ctx context.Context, userID, planID int64) (*PlanResponse, error) {
	now := s.now().UTC()
	res := s.db.WithContext(ctx).Model(&schemas.Plan{}).
		Where("id = ? AND user_id = ? AND state = ?", planID, userID, schemas.PlanProposed).
		Updates(map[string]any{"state": schemas.PlanDismissed, "expires_at": now.Add(RetentionTTL), "updated_at": now})
	if res.Error != nil {
		return nil, errors.Internal("failed to dismiss the plan", res.Error)
	}
	plan, err := s.load(s.db.WithContext(ctx), userID, planID, false)
	if err != nil {
		return nil, err
	}
	if plan.State == schemas.PlanApplied {
		return nil, errors.Failed("this plan is already applied")
	}
	return s.respond(*plan), nil
}

// Apply puts the chosen blocks of a proposed plan on the calendars as
// events, all in one transaction, and marks the plan applied. Blocks not
// chosen stay unapplied. Without force, a block that now overlaps an event is
// refused and nothing is created. Each event's description names the task
// link and the reason.
func (s *Service) Apply(ctx context.Context, userID, planID int64, req ApplyRequest) (*ApplyResponse, error) {
	now := s.now().UTC()
	plan, err := s.load(s.db.WithContext(ctx), userID, planID, false)
	if err != nil {
		return nil, err
	}
	if err := applicable(plan, now); err != nil {
		return nil, err
	}
	chosen, err := choose(plan, req.BlockIDs)
	if err != nil {
		return nil, err
	}
	items := make([]events.NewEvent, len(chosen))
	blocks := make([]BlockInput, len(chosen))
	for i, b := range chosen {
		if b.StartAt.Before(now) {
			return nil, errors.Failed(fmt.Sprintf("block %d starts in the past. Ask for a new plan", b.Position+1))
		}
		blocks[i] = BlockInput{StartAt: b.StartAt, EndAt: b.EndAt}
		items[i] = events.NewEvent{CalendarID: b.CalendarID, Request: eventOf(plan, b)}
	}
	if !req.Force {
		if err := s.refuseClashes(ctx, userID, chosen, blocks); err != nil {
			return nil, err
		}
	}
	created, err := s.events.CreateBatch(ctx, userID, items, func(tx *gorm.DB, ids []int64) error {
		return s.record(tx, userID, planID, chosen, ids, now)
	})
	if err != nil {
		return nil, err
	}
	plan, err = s.load(s.db.WithContext(ctx), userID, planID, false)
	if err != nil {
		return nil, err
	}
	return &ApplyResponse{Plan: *s.respond(*plan), Events: created}, nil
}

func (s *Service) refuseClashes(ctx context.Context, userID int64, chosen []schemas.PlanBlock, blocks []BlockInput) error {
	hits, err := s.clashes(ctx, userID, blocks)
	if err != nil || len(hits) == 0 {
		return err
	}
	ids := make([]int, len(hits))
	for i, h := range hits {
		ids[i] = int(chosen[h-1].ID)
	}
	return errors.Conflict("Blocks now overlapping an event (block ids " + joined(ids) + "). Apply again with force to keep them, or leave them out")
}

// applicable refuses a plan that is no longer proposed or has expired.
func applicable(plan *schemas.Plan, now time.Time) error {
	switch {
	case plan.State != schemas.PlanProposed:
		return errors.Failed("this plan is already " + plan.State)
	case !plan.ExpiresAt.After(now):
		return errors.Failed("this plan expired. Ask for a new one")
	}
	return nil
}

// choose is the blocks to apply: the ones named, or all when none is.
func choose(plan *schemas.Plan, ids []int64) ([]schemas.PlanBlock, error) {
	if len(ids) == 0 {
		return plan.Blocks, nil
	}
	var out []schemas.PlanBlock
	for _, b := range plan.Blocks {
		if slices.Contains(ids, b.ID) {
			out = append(out, b)
		}
	}
	if len(out) != len(slices.Compact(slices.Sorted(slices.Values(ids)))) {
		return nil, errors.Invalid("block_ids names a block that is not in this plan")
	}
	return out, nil
}

// eventOf is the event a block becomes. The description starts with the task
// link, then the reason, both as the plain text they were saved as.
func eventOf(plan *schemas.Plan, b schemas.PlanBlock) events.CreateEventRequest {
	var lines []string
	switch {
	case b.TaskURL != "":
		lines = append(lines, "Task: "+b.TaskURL)
	case b.TaskRef != "":
		lines = append(lines, "Task: "+b.TaskRef)
	}
	if b.Reason != "" {
		lines = append(lines, b.Reason)
	}
	return events.CreateEventRequest{
		Title: b.Title, Description: strings.Join(lines, "\n\n"), StartAt: b.StartAt, EndAt: b.EndAt,
		Timezone: plan.Timezone, Status: "confirmed",
	}
}

// record runs inside the events transaction: it locks the plan, checks it is
// still proposed so two applies cannot both win, links each applied block to
// its event and marks the plan applied.
func (s *Service) record(tx *gorm.DB, userID, planID int64, chosen []schemas.PlanBlock, eventIDs []int64, now time.Time) error {
	plan, err := s.load(tx, userID, planID, true)
	if err != nil {
		return err
	}
	if err := applicable(plan, now); err != nil {
		return err
	}
	for i, b := range chosen {
		if err := tx.Model(&schemas.PlanBlock{}).Where("id = ?", b.ID).Update("event_id", eventIDs[i]).Error; err != nil {
			return errors.Internal("failed to record the applied block", err)
		}
	}
	err = tx.Model(&schemas.Plan{}).Where("id = ?", planID).
		Updates(map[string]any{"state": schemas.PlanApplied, "expires_at": now.Add(RetentionTTL), "updated_at": now}).Error
	if err != nil {
		return errors.Internal("failed to mark the plan applied", err)
	}
	return nil
}
