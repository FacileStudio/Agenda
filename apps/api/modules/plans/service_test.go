package plans

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/modules/events"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

func TestProposeRefusals(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	reader := schemas.Calendar{OwnerID: h.other.ID, Slug: "shared", Name: "Shared"}
	h.db.Create(&reader)
	h.db.Create(&schemas.CalendarMember{CalendarID: reader.ID, UserID: h.owner.ID, Role: "reader"})
	h.busy(9, 10)
	onForeign, onReader, onMissing := h.block(11, 12), h.block(11, 12), h.block(11, 12)
	onForeign.CalendarID, onReader.CalendarID, onMissing.CalendarID = h.foreign.ID, reader.ID, 9999
	cases := []struct {
		name   string
		blocks []BlockInput
		want   int
		hint   string
	}{
		{"overlaps a meeting", []BlockInput{h.block(9, 11)}, http.StatusConflict, "Blocks overlapping an existing event: 1"},
		{"another person's calendar", []BlockInput{onForeign}, http.StatusNotFound, "calendar"},
		{"read-only calendar", []BlockInput{onReader}, http.StatusForbidden, "read-only"},
		{"unknown calendar", []BlockInput{onMissing}, http.StatusNotFound, "calendar 9999 not found"},
		{"free time", []BlockInput{h.block(10, 11)}, http.StatusOK, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := h.service.Propose(ctx, h.owner.ID, proposal(c.blocks...))
			if status(err) != c.want || (c.hint != "" && !strings.Contains(err.Error(), c.hint)) {
				t.Fatalf("want %d holding %q, got %v", c.want, c.hint, err)
			}
		})
	}
}

func TestProposeIgnoresAllDayAndCancelledEvents(t *testing.T) {
	h := newHarness(t)
	b := h.block(9, 10)
	for _, req := range []struct {
		allDay bool
		status string
	}{{true, ""}, {false, "cancelled"}} {
		h.db.Create(&schemas.Event{
			CalendarID: h.calendar.ID, UID: strings.Repeat("x", 3) + req.status, Title: "Soft", StartAt: b.StartAt, EndAt: b.EndAt,
			IsAllDay: req.allDay, Status: req.status,
		})
	}
	h.propose(b)
}

func TestProposeChangesNoCalendarAndKeepsSpace(t *testing.T) {
	h := newHarness(t)
	space := schemas.Space{Name: "Studio"}
	if err := h.db.Create(&space).Error; err != nil {
		t.Fatal(err)
	}
	h.db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: h.owner.ID, Role: "member"})
	shared := schemas.Calendar{OwnerID: 0, SpaceID: &space.ID, Slug: "space-1", Name: "Studio"}
	if err := h.db.Create(&shared).Error; err != nil {
		t.Fatal(err)
	}
	inSpace := h.block(11, 12)
	inSpace.CalendarID = shared.ID
	plan := h.propose(inSpace, inSpace2(inSpace))
	if plan.SpaceID == nil || *plan.SpaceID != space.ID || plan.State != schemas.PlanProposed || h.eventCount() != 0 {
		t.Fatalf("want a proposed plan in the space and no event, got %+v with %d events", plan, h.eventCount())
	}
	mixed := h.propose(h.block(9, 10), inSpace)
	if mixed.SpaceID != nil {
		t.Fatalf("blocks in different calendars have no single space, got %v", *mixed.SpaceID)
	}
}

func inSpace2(b BlockInput) BlockInput {
	b.StartAt, b.EndAt = b.StartAt.Add(2*time.Hour), b.EndAt.Add(2*time.Hour)
	return b
}

func TestProposeCapsOpenPlans(t *testing.T) {
	h := newHarness(t)
	for range MaxOpenPlans {
		h.propose(h.block(9, 10))
	}
	_, err := h.service.Propose(context.Background(), h.owner.ID, proposal(h.block(9, 10)))
	if status(err) != http.StatusPreconditionFailed {
		t.Fatalf("want 412 at the cap, got %v", err)
	}
	first, _ := h.service.List(context.Background(), h.owner.ID, "", 50)
	if _, err := h.service.Dismiss(context.Background(), h.owner.ID, first[0].ID); err != nil {
		t.Fatal(err)
	}
	h.propose(h.block(9, 10))
}

func TestExpiredPlansLapseAndArePurged(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	old := h.propose(h.block(9, 10))
	h.now = h.now.Add(ProposedTTL + time.Minute)
	if got, _ := h.service.List(ctx, h.owner.ID, "", 10); len(got) != 0 {
		t.Fatalf("an expired plan must not be listed, got %d", len(got))
	}
	got, err := h.service.Get(ctx, h.owner.ID, old.ID)
	if err != nil || !got.Expired {
		t.Fatalf("an expired plan reads as expired, got %+v %v", got, err)
	}
	if _, err := h.service.Apply(ctx, h.owner.ID, old.ID, ApplyRequest{}); status(err) != http.StatusPreconditionFailed || h.eventCount() != 0 {
		t.Fatalf("want 412 and no event for an expired plan, got %v", err)
	}
	fresh := h.block(9, 10)
	fresh.StartAt, fresh.EndAt = h.now.Add(time.Hour), h.now.Add(2*time.Hour)
	h.propose(fresh)
	var rows int64
	h.db.Model(&schemas.Plan{}).Where("id = ?", old.ID).Count(&rows)
	if rows != 0 {
		t.Fatal("a new proposal must purge expired plans")
	}
}

func TestApply(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	plan := h.propose(h.block(9, 10), h.block(11, 12), h.block(14, 15))
	ids := []int64{plan.Blocks[0].ID, plan.Blocks[2].ID}
	got, err := h.service.Apply(ctx, h.owner.ID, plan.ID, ApplyRequest{BlockIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Events) != 2 || got.Plan.State != schemas.PlanApplied || h.eventCount() != 2 {
		t.Fatalf("want 2 events and an applied plan, got %+v", got)
	}
	first := got.Events[0]
	if first.Title != "Write the spec" || first.CalendarID != h.calendar.ID || !first.StartAt.Equal(plan.Blocks[0].StartAt) {
		t.Fatalf("first event %+v", first)
	}
	if !strings.Contains(first.Description, "Task: https://opus.example/tasks/9") || !strings.Contains(first.Description, "Due Friday") {
		t.Fatalf("the event must carry the task link and the reason, got %q", first.Description)
	}
	if got.Plan.Blocks[0].EventID == nil || got.Plan.Blocks[1].EventID != nil || got.Plan.Blocks[2].EventID == nil {
		t.Fatalf("only the chosen blocks link to an event: %+v", got.Plan.Blocks)
	}
	if _, err := h.service.Apply(ctx, h.owner.ID, plan.ID, ApplyRequest{}); status(err) != http.StatusPreconditionFailed || h.eventCount() != 2 {
		t.Fatalf("a plan applies once, got %v", err)
	}
	if _, err := h.service.Dismiss(ctx, h.owner.ID, plan.ID); status(err) != http.StatusPreconditionFailed {
		t.Fatalf("an applied plan cannot be dismissed, got %v", err)
	}
}

func TestApplyAllWhenNoBlockIsNamed(t *testing.T) {
	h := newHarness(t)
	plan := h.propose(h.block(9, 10), h.block(11, 12))
	got, err := h.service.Apply(context.Background(), h.owner.ID, plan.ID, ApplyRequest{})
	if err != nil || len(got.Events) != 2 {
		t.Fatalf("want both blocks applied, got %+v %v", got, err)
	}
}

func TestApplyRefusals(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	plan := h.propose(h.block(9, 10), h.block(11, 12))
	other := h.propose(h.block(15, 16))
	cases := []struct {
		name string
		user int64
		id   int64
		req  ApplyRequest
		want int
	}{
		{"wrong owner", h.other.ID, plan.ID, ApplyRequest{}, http.StatusNotFound},
		{"unknown plan", h.owner.ID, 99999, ApplyRequest{}, http.StatusNotFound},
		{"block of another plan", h.owner.ID, plan.ID, ApplyRequest{BlockIDs: []int64{other.Blocks[0].ID}}, http.StatusBadRequest},
		{"one good, one unknown block", h.owner.ID, plan.ID, ApplyRequest{BlockIDs: []int64{plan.Blocks[0].ID, 99999}}, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := h.service.Apply(ctx, c.user, c.id, c.req); status(err) != c.want {
				t.Fatalf("want %d, got %v", c.want, err)
			}
			if h.eventCount() != 0 {
				t.Fatal("a refused apply must create nothing")
			}
		})
	}
}

func TestApplyRefusesBusyTimeUnlessForced(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	plan := h.propose(h.block(9, 10), h.block(11, 12))
	h.busy(11, 12)
	_, err := h.service.Apply(ctx, h.owner.ID, plan.ID, ApplyRequest{})
	if status(err) != http.StatusConflict || !strings.Contains(err.Error(), "block ids") || h.eventCount() != 1 {
		t.Fatalf("want 409 with nothing created beyond the meeting, got %v", err)
	}
	got, err := h.service.Apply(ctx, h.owner.ID, plan.ID, ApplyRequest{BlockIDs: []int64{plan.Blocks[0].ID}})
	if err != nil || len(got.Events) != 1 {
		t.Fatalf("leaving the busy block out must work, got %v", err)
	}
	again := h.propose(h.block(14, 15), h.block(16, 17))
	h.busy(14, 15)
	forced, err := h.service.Apply(ctx, h.owner.ID, again.ID, ApplyRequest{Force: true})
	if err != nil || len(forced.Events) != 2 {
		t.Fatalf("force must keep the overlapping block, got %v", err)
	}
}

func TestApplyIsAtomic(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	plan := h.propose(h.block(9, 10), h.block(11, 12))
	h.db.Model(&schemas.PlanBlock{}).Where("id = ?", plan.Blocks[1].ID).Update("calendar_id", h.foreign.ID)
	if _, err := h.service.Apply(ctx, h.owner.ID, plan.ID, ApplyRequest{}); status(err) == http.StatusOK {
		t.Fatal("a block on a calendar the person cannot write to must fail the apply")
	}
	got, _ := h.service.Get(ctx, h.owner.ID, plan.ID)
	if h.eventCount() != 0 || got.State != schemas.PlanProposed {
		t.Fatalf("nothing may be created and the plan stays proposed, got %d events and %q", h.eventCount(), got.State)
	}
}

func TestApplyRefusesAStartedBlock(t *testing.T) {
	h := newHarness(t)
	plan := h.propose(h.block(9, 10))
	h.now = plan.Blocks[0].StartAt.Add(time.Minute)
	if _, err := h.service.Apply(context.Background(), h.owner.ID, plan.ID, ApplyRequest{}); status(err) != http.StatusPreconditionFailed {
		t.Fatalf("want 412 for a block already started, got %v", err)
	}
}

func TestDismiss(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	plan := h.propose(h.block(9, 10))
	if _, err := h.service.Dismiss(ctx, h.other.ID, plan.ID); status(err) != http.StatusNotFound {
		t.Fatalf("another person cannot dismiss, got %v", err)
	}
	for range 2 {
		got, err := h.service.Dismiss(ctx, h.owner.ID, plan.ID)
		if err != nil || got.State != schemas.PlanDismissed {
			t.Fatalf("dismiss twice is fine, got %+v %v", got, err)
		}
	}
	if _, err := h.service.Apply(ctx, h.owner.ID, plan.ID, ApplyRequest{}); status(err) != http.StatusPreconditionFailed || h.eventCount() != 0 {
		t.Fatalf("a dismissed plan cannot be applied, got %v", err)
	}
}

func TestListFiltersByStateAndOwner(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	kept := h.propose(h.block(9, 10))
	gone := h.propose(h.block(11, 12))
	h.service.Dismiss(ctx, h.owner.ID, gone.ID)
	proposed, err := h.service.List(ctx, h.owner.ID, schemas.PlanProposed, 0)
	if err != nil || len(proposed) != 1 || proposed[0].ID != kept.ID {
		t.Fatalf("want the proposed plan only, got %+v %v", proposed, err)
	}
	if all, _ := h.service.List(ctx, h.owner.ID, "", 0); len(all) != 2 || all[0].ID != gone.ID {
		t.Fatalf("want both, newest first, got %+v", all)
	}
	if theirs, _ := h.service.List(ctx, h.other.ID, "", 0); len(theirs) != 0 {
		t.Fatalf("another person sees nothing, got %+v", theirs)
	}
	if _, err := h.service.List(ctx, h.owner.ID, "bogus", 0); status(err) != http.StatusBadRequest {
		t.Fatalf("want 400 for an unknown state, got %v", err)
	}
}

func TestBatchRollsBackWhenTheRecordFails(t *testing.T) {
	h := newHarness(t)
	b := h.block(9, 10)
	items := []events.NewEvent{
		{CalendarID: h.calendar.ID, Request: events.CreateEventRequest{Title: "A", StartAt: b.StartAt, EndAt: b.EndAt}},
		{CalendarID: h.calendar.ID, Request: events.CreateEventRequest{Title: "B", StartAt: b.StartAt, EndAt: b.EndAt}},
	}
	refusal := errors.Failed("plan changed")
	_, err := h.events.CreateBatch(context.Background(), h.owner.ID, items, func(*gorm.DB, []int64) error { return refusal })
	if err != refusal || h.eventCount() != 0 {
		t.Fatalf("want the refusal unchanged and no event, got %v with %d events", err, h.eventCount())
	}
}

func TestTwoAppliesCreateTheEventsOnce(t *testing.T) {
	h := newHarness(t)
	plan := h.propose(h.block(9, 10), h.block(11, 12))
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := h.service.Apply(context.Background(), h.owner.ID, plan.ID, ApplyRequest{Force: true})
			results <- err
		}()
	}
	wins := 0
	for range 2 {
		if err := <-results; err == nil {
			wins++
		}
	}
	if wins != 1 || h.eventCount() != 2 {
		t.Fatalf("want one winner and 2 events, got %d winners and %d events", wins, h.eventCount())
	}
}
