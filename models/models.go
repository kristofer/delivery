package models

import (
	"database/sql"
	"time"
)

// User roles.
const (
	RoleAdmin      = "admin"
	RoleInstructor = "instructor"
	RoleStudent    = "student"
)

// User represents a GitHub-authenticated user.
type User struct {
	ID          int64
	GitHubLogin string
	GitHubID    sql.NullInt64
	Name        string
	Email       string
	AvatarURL   string
	Role        string
	Whitelisted bool
	CreatedAt   time.Time
}

// LocalAdmin is a locally-created admin (username + bcrypt password).
type LocalAdmin struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

// Cohort is a group of students working in a particular focus area.
type Cohort struct {
	ID        int64
	Name      string // e.g. "Summer26"
	Focus     string // e.g. "Java", "Data"
	StartDate sql.NullTime
	EndDate   sql.NullTime
	CreatedAt time.Time
}

// Week is a single week within a Cohort.
type Week struct {
	ID         int64
	CohortID   int64
	WeekNumber int
	Title      string
	StartDate  sql.NullTime
}

// Activity is a single programming or research assignment.
type Activity struct {
	ID           int64
	WeekID       int64
	Title        string
	Description  string
	SourceURL    string
	AssignedDate sql.NullTime
	DueDate      sql.NullTime
	IsGroup      bool
	CreatedBy    sql.NullInt64
	CreatedAt    time.Time
}

// Group is a set of students collaborating on a group Activity.
type Group struct {
	ID         int64
	ActivityID int64
	Name       string
	CreatedAt  time.Time
	Members    []User
}

// Submission is a student (or group) repo URL submitted for an Activity.
type Submission struct {
	ID          int64
	ActivityID  int64
	UserID      sql.NullInt64
	GroupID     sql.NullInt64
	RepoURL     string
	Notes       string
	SubmittedAt time.Time
	UpdatedAt   time.Time

	// Joined fields (populated by queries)
	User     *User
	Group    *Group
	Activity *Activity
}
