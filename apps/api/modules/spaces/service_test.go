package spaces

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/testdb"

	"gorm.io/gorm"
)

const testBootstrap = "docker compose up -d db   # or any PostgreSQL 16"

var (
	testDBOnce sync.Once
	testDB     *gorm.DB
	testDBErr  error
	testDBCfg  = testdb.Config{Prefix: "agenda_spaces_test", Migrate: schemas.Migrate}
)

func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	url, configured := testdb.URL()
	if !configured {
		testdb.Announce(testBootstrap)
		t.Skip(testdb.SkipReason(testBootstrap))
	}

	testDBOnce.Do(func() { testDB, testDBErr = testdb.Open(url, testDBCfg) })
	if testDBErr != nil {
		t.Fatalf("test database: %v", testDBErr)
	}
	if err := testdb.Truncate(testDB, testDBCfg); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	return testDB
}

func newUser(t *testing.T, db *gorm.DB, email string) int64 {
	t.Helper()
	user := schemas.User{Email: email, Name: email}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return user.ID
}

// spaceWith builds a space and puts each user in it at the given role.
func spaceWith(t *testing.T, db *gorm.DB, roles map[int64]string) int64 {
	t.Helper()
	space := schemas.Space{Name: "Studio"}
	if err := db.Create(&space).Error; err != nil {
		t.Fatalf("create space: %v", err)
	}
	for userID, role := range roles {
		member := schemas.SpaceMember{SpaceID: space.ID, UserID: userID, Role: role}
		if err := db.Create(&member).Error; err != nil {
			t.Fatalf("create membership: %v", err)
		}
	}
	return space.ID
}

func roleOf(t *testing.T, db *gorm.DB, spaceID, userID int64) string {
	t.Helper()
	var member schemas.SpaceMember
	err := db.Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error
	if err != nil {
		return ""
	}
	return member.Role
}

// AddMember checked that the actor was an owner or an admin and then wrote
// whatever role the body asked for, so an admin could pass an existing
// member's address with role "owner" and promote them. The write is an upsert,
// which is what turns "add" into "promote": UpdateMemberRole is owner-only and
// this walked straight around it. Self-promotion was already blocked by the
// target.ID == userID check, so the escalation needed an accomplice — the
// admin promotes a confederate, who then removes the admin's limits.
// TestAnAdminCannotPromoteAnotherMemberToOwner is the regression.
func TestAnAdminCannotPromoteAnotherMemberToOwner(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db)

	owner := newUser(t, db, "owner@facile.studio")
	admin := newUser(t, db, "admin@facile.studio")
	accomplice := newUser(t, db, "accomplice@facile.studio")
	spaceID := spaceWith(t, db, map[int64]string{
		owner:      "owner",
		admin:      "admin",
		accomplice: "member",
	})

	err := service.AddMember(context.Background(), admin, spaceID, &AddMemberRequest{
		Email: "accomplice@facile.studio",
		Role:  "owner",
	})
	if err == nil {
		t.Fatal("an admin granted ownership, which is the escalation UpdateMemberRole is owner-only to prevent")
	}
	if !strings.Contains(err.Error(), "rank") && !strings.Contains(err.Error(), "permission") {
		t.Fatalf("unexpected error: %v", err)
	}
	if role := roleOf(t, db, spaceID, accomplice); role != "member" {
		t.Fatalf("the accomplice's role changed to %q", role)
	}
}

// The same upsert let an admin hand out any role at all, so the rank rule has
// to hold for a brand new member too — otherwise the escalation is one
// invitation away instead of one promotion away.
func TestAnAdminCannotInviteANewOwner(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db)

	owner := newUser(t, db, "owner@facile.studio")
	admin := newUser(t, db, "admin@facile.studio")
	outsider := newUser(t, db, "outsider@facile.studio")
	spaceID := spaceWith(t, db, map[int64]string{owner: "owner", admin: "admin"})

	err := service.AddMember(context.Background(), admin, spaceID, &AddMemberRequest{
		Email: "outsider@facile.studio",
		Role:  "owner",
	})
	if err == nil {
		t.Fatal("an admin invited a new owner")
	}
	if role := roleOf(t, db, spaceID, outsider); role != "" {
		t.Fatalf("the outsider joined anyway, as %q", role)
	}
}

// The rank rule is a ceiling, not a freeze. An admin still runs the space
// day to day, and an owner is still allowed to appoint a second owner.
func TestGrantingAtOrBelowYourOwnRankStillWorks(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db)

	owner := newUser(t, db, "owner@facile.studio")
	admin := newUser(t, db, "admin@facile.studio")
	newcomer := newUser(t, db, "newcomer@facile.studio")
	successor := newUser(t, db, "successor@facile.studio")
	spaceID := spaceWith(t, db, map[int64]string{owner: "owner", admin: "admin"})

	if err := service.AddMember(context.Background(), admin, spaceID, &AddMemberRequest{
		Email: "newcomer@facile.studio",
		Role:  "admin",
	}); err != nil {
		t.Fatalf("an admin could not appoint another admin: %v", err)
	}
	if role := roleOf(t, db, spaceID, newcomer); role != "admin" {
		t.Fatalf("newcomer role: got %q want admin", role)
	}

	if err := service.AddMember(context.Background(), owner, spaceID, &AddMemberRequest{
		Email: "successor@facile.studio",
		Role:  "owner",
	}); err != nil {
		t.Fatalf("an owner could not appoint a second owner: %v", err)
	}
	if role := roleOf(t, db, spaceID, successor); role != "owner" {
		t.Fatalf("successor role: got %q want owner", role)
	}
}

// A space with no owner cannot be deleted and cannot have its roles changed,
// because both are owner-only. RemoveMember refused every owner to a non-owner
// and no owner at all to an owner, so the one removal that actually strands
// the space — an owner removing the last one — was the one it allowed.
// TestTheSoleOwnerCannotBeRemoved checks the rule is the count, not the role.
func TestTheSoleOwnerCannotBeRemoved(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db)

	owner := newUser(t, db, "owner@facile.studio")
	spaceID := spaceWith(t, db, map[int64]string{owner: "owner"})

	if err := service.RemoveMember(context.Background(), owner, spaceID, owner); err == nil {
		t.Fatal("the sole owner was removed, leaving a space nobody can administer")
	}
	if role := roleOf(t, db, spaceID, owner); role != "owner" {
		t.Fatal("the sole owner is gone")
	}
}

// With a second owner in place the space keeps an administrator either way, so
// the refusal above must lift rather than becoming a rule that owners are
// permanent.
func TestOneOfTwoOwnersCanBeRemoved(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db)

	owner := newUser(t, db, "owner@facile.studio")
	second := newUser(t, db, "second@facile.studio")
	spaceID := spaceWith(t, db, map[int64]string{owner: "owner", second: "owner"})

	if err := service.RemoveMember(context.Background(), owner, spaceID, second); err != nil {
		t.Fatalf("one of two owners could not be removed: %v", err)
	}
	if role := roleOf(t, db, spaceID, second); role != "" {
		t.Fatalf("the second owner is still a member, as %q", role)
	}
}

// An admin acting on an owner is acting above their own rank, which is the
// same rule the grant ceiling enforces from the other direction.
func TestAnAdminCannotRemoveAnOwner(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db)

	owner := newUser(t, db, "owner@facile.studio")
	second := newUser(t, db, "second@facile.studio")
	admin := newUser(t, db, "admin@facile.studio")
	spaceID := spaceWith(t, db, map[int64]string{owner: "owner", second: "owner", admin: "admin"})

	if err := service.RemoveMember(context.Background(), admin, spaceID, second); err == nil {
		t.Fatal("an admin removed an owner")
	}
	if role := roleOf(t, db, spaceID, second); role != "owner" {
		t.Fatalf("the owner's membership changed to %q", role)
	}
}

// LeaveSpace already refused only the sole owner. It is pinned here so the
// removal fix above does not get "harmonised" into the stricter rule by
// somebody who reads the two paths side by side.
func TestAnOwnerCanLeaveWhenAnotherOwnerRemains(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db)

	owner := newUser(t, db, "owner@facile.studio")
	second := newUser(t, db, "second@facile.studio")
	spaceID := spaceWith(t, db, map[int64]string{owner: "owner", second: "owner"})

	if err := service.LeaveSpace(context.Background(), second, spaceID); err != nil {
		t.Fatalf("an owner could not leave a space with two owners: %v", err)
	}
	if err := service.LeaveSpace(context.Background(), owner, spaceID); err == nil {
		t.Fatal("the last owner left, stranding the space")
	}
}
