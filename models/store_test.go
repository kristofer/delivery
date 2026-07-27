package models_test

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/kristofer/delivery/db"
	"github.com/kristofer/delivery/models"
)

func TestUserStore(t *testing.T) {
	f, err := os.CreateTemp("", "deliver-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	database, err := db.Open(f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	store := &models.UserStore{DB: database}

	// CreateLocalAdmin
	if err := store.CreateLocalAdmin("admin", "hashed"); err != nil {
		t.Fatalf("CreateLocalAdmin: %v", err)
	}

	// GetLocalAdmin
	a, err := store.GetLocalAdmin("admin")
	if err != nil {
		t.Fatalf("GetLocalAdmin: %v", err)
	}
	if a.Username != "admin" {
		t.Errorf("expected username=admin, got %q", a.Username)
	}

	// Count
	n, err := store.CountLocalAdmins()
	if err != nil {
		t.Fatalf("CountLocalAdmins: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 local admin, got %d", n)
	}

	// UpsertGitHubUser
	u, err := store.UpsertGitHubUser("student1", 1001, "Alice", "alice@example.com", "")
	if err != nil {
		t.Fatalf("UpsertGitHubUser: %v", err)
	}
	if u.GitHubLogin != "student1" {
		t.Errorf("expected github_login=student1, got %q", u.GitHubLogin)
	}
	if u.Role != models.RoleStudent {
		t.Errorf("expected role=student, got %q", u.Role)
	}
	if u.Whitelisted {
		t.Error("new user should not be whitelisted by default")
	}

	// WhitelistUser
	if err := store.WhitelistUser(u.ID, models.RoleInstructor); err != nil {
		t.Fatalf("WhitelistUser: %v", err)
	}
	u2, _ := store.GetUserByID(u.ID)
	if !u2.Whitelisted {
		t.Error("user should be whitelisted after WhitelistUser")
	}
	if u2.Role != models.RoleInstructor {
		t.Errorf("expected role=instructor, got %q", u2.Role)
	}

	// ListUsers
	users, err := store.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}

	// RevokeUser
	if err := store.RevokeUser(u.ID); err != nil {
		t.Fatalf("RevokeUser: %v", err)
	}
	u3, _ := store.GetUserByID(u.ID)
	if u3.Whitelisted {
		t.Error("user should not be whitelisted after RevokeUser")
	}
}

func TestCohortStore(t *testing.T) {
	f, _ := os.CreateTemp("", "deliver-test-*.db")
	f.Close()
	defer os.Remove(f.Name())

	database, _ := db.Open(f.Name())
	defer database.Close()

	cs := &models.CohortStore{DB: database}

	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	c, err := cs.CreateCohort("Summer26", "Java", &start, &end)
	if err != nil {
		t.Fatalf("CreateCohort: %v", err)
	}
	if c.Name != "Summer26" {
		t.Errorf("expected name=Summer26, got %q", c.Name)
	}

	list, err := cs.ListCohorts()
	if err != nil {
		t.Fatalf("ListCohorts: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 cohort, got %d", len(list))
	}

	if err := cs.UpdateCohort(c.ID, "Summer26", "Data", nil, nil); err != nil {
		t.Fatalf("UpdateCohort: %v", err)
	}
	c2, _ := cs.GetCohort(c.ID)
	if c2.Focus != "Data" {
		t.Errorf("expected focus=Data after update, got %q", c2.Focus)
	}

	if err := cs.DeleteCohort(c.ID); err != nil {
		t.Fatalf("DeleteCohort: %v", err)
	}
	list2, _ := cs.ListCohorts()
	if len(list2) != 0 {
		t.Errorf("expected 0 cohorts after delete, got %d", len(list2))
	}
}

func TestWeekAndActivityStore(t *testing.T) {
	f, _ := os.CreateTemp("", "deliver-test-*.db")
	f.Close()
	defer os.Remove(f.Name())

	database, _ := db.Open(f.Name())
	defer database.Close()

	cs := &models.CohortStore{DB: database}
	ws := &models.WeekStore{DB: database}
	as := &models.ActivityStore{DB: database}

	cohort, _ := cs.CreateCohort("Fall26", "Java", nil, nil)

	week, err := ws.CreateWeek(cohort.ID, 1, "Week 1", nil)
	if err != nil {
		t.Fatalf("CreateWeek: %v", err)
	}
	if week.WeekNumber != 1 {
		t.Errorf("expected week_number=1, got %d", week.WeekNumber)
	}

	weeks, _ := ws.ListWeeksByCohort(cohort.ID)
	if len(weeks) != 1 {
		t.Errorf("expected 1 week, got %d", len(weeks))
	}

	act, err := as.CreateActivity(models.Activity{
		WeekID:      week.ID,
		Title:       "Hello World",
		Description: "Write hello world",
		SourceURL:   "https://github.com/example/template",
		IsGroup:     false,
	})
	if err != nil {
		t.Fatalf("CreateActivity: %v", err)
	}
	if act.Title != "Hello World" {
		t.Errorf("expected title=Hello World, got %q", act.Title)
	}

	acts, _ := as.ListActivitiesByWeek(week.ID)
	if len(acts) != 1 {
		t.Errorf("expected 1 activity, got %d", len(acts))
	}

	act.Title = "Updated Title"
	if err := as.UpdateActivity(*act); err != nil {
		t.Fatalf("UpdateActivity: %v", err)
	}
	act2, _ := as.GetActivity(act.ID)
	if act2.Title != "Updated Title" {
		t.Errorf("expected updated title, got %q", act2.Title)
	}
}

func TestSubmissionStore(t *testing.T) {
	f, _ := os.CreateTemp("", "deliver-test-*.db")
	f.Close()
	defer os.Remove(f.Name())

	database, _ := db.Open(f.Name())
	defer database.Close()

	us := &models.UserStore{DB: database}
	cs := &models.CohortStore{DB: database}
	ws := &models.WeekStore{DB: database}
	as := &models.ActivityStore{DB: database}
	ss := &models.SubmissionStore{DB: database}

	// Setup
	user, _ := us.UpsertGitHubUser("student1", 1001, "Alice", "alice@example.com", "")
	cohort, _ := cs.CreateCohort("Summer26", "Java", nil, nil)
	week, _ := ws.CreateWeek(cohort.ID, 1, "Week 1", nil)
	activity, _ := as.CreateActivity(models.Activity{WeekID: week.ID, Title: "HW1"})

	sub, err := ss.CreateSubmission(models.Submission{
		ActivityID: activity.ID,
		UserID:     sql.NullInt64{Int64: user.ID, Valid: true},
		RepoURL:    "https://github.com/student1/hw1",
	})
	if err != nil {
		t.Fatalf("CreateSubmission: %v", err)
	}
	if sub.RepoURL != "https://github.com/student1/hw1" {
		t.Errorf("expected repo_url, got %q", sub.RepoURL)
	}

	subs, _ := ss.ListSubmissionsByActivity(activity.ID)
	if len(subs) != 1 {
		t.Errorf("expected 1 submission, got %d", len(subs))
	}

	if err := ss.UpdateSubmission(sub.ID, "https://github.com/student1/hw1-v2", "updated"); err != nil {
		t.Fatalf("UpdateSubmission: %v", err)
	}
	sub2, _ := ss.GetSubmission(sub.ID)
	if sub2.RepoURL != "https://github.com/student1/hw1-v2" {
		t.Errorf("expected updated URL, got %q", sub2.RepoURL)
	}
}
