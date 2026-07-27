package models

import (
	"database/sql"
	"fmt"
	"time"
)

// UserStore handles database operations for User and LocalAdmin.
type UserStore struct {
	DB *sql.DB
}

// CreateLocalAdmin inserts a new local admin record.
func (s *UserStore) CreateLocalAdmin(username, passwordHash string) error {
	_, err := s.DB.Exec(
		`INSERT INTO local_admins (username, password_hash) VALUES (?, ?)`,
		username, passwordHash,
	)
	return err
}

// GetLocalAdmin retrieves a local admin by username.
func (s *UserStore) GetLocalAdmin(username string) (*LocalAdmin, error) {
	row := s.DB.QueryRow(
		`SELECT id, username, password_hash, created_at FROM local_admins WHERE username = ?`,
		username,
	)
	a := &LocalAdmin{}
	if err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.CreatedAt); err != nil {
		return nil, err
	}
	return a, nil
}

// CountLocalAdmins returns the number of local admin accounts.
func (s *UserStore) CountLocalAdmins() (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM local_admins`).Scan(&n)
	return n, err
}

// UpsertGitHubUser creates or updates a GitHub-authenticated user.
func (s *UserStore) UpsertGitHubUser(login string, githubID int64, name, email, avatarURL string) (*User, error) {
	_, err := s.DB.Exec(`
		INSERT INTO users (github_login, github_id, name, email, avatar_url)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(github_login) DO UPDATE SET
			github_id  = excluded.github_id,
			name       = excluded.name,
			email      = excluded.email,
			avatar_url = excluded.avatar_url`,
		login, githubID, name, email, avatarURL,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert github user: %w", err)
	}
	return s.GetUserByLogin(login)
}

// GetUserByLogin retrieves a user by GitHub login.
func (s *UserStore) GetUserByLogin(login string) (*User, error) {
	row := s.DB.QueryRow(
		`SELECT id, github_login, github_id, name, email, avatar_url, role, whitelisted, created_at
		 FROM users WHERE github_login = ?`, login,
	)
	return scanUser(row)
}

// GetUserByID retrieves a user by primary key.
func (s *UserStore) GetUserByID(id int64) (*User, error) {
	row := s.DB.QueryRow(
		`SELECT id, github_login, github_id, name, email, avatar_url, role, whitelisted, created_at
		 FROM users WHERE id = ?`, id,
	)
	return scanUser(row)
}

// ListUsers returns all users.
func (s *UserStore) ListUsers() ([]User, error) {
	rows, err := s.DB.Query(
		`SELECT id, github_login, github_id, name, email, avatar_url, role, whitelisted, created_at
		 FROM users ORDER BY github_login`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// WhitelistUser sets a user's whitelisted flag and role.
func (s *UserStore) WhitelistUser(id int64, role string) error {
	_, err := s.DB.Exec(
		`UPDATE users SET whitelisted = 1, role = ? WHERE id = ?`,
		role, id,
	)
	return err
}

// RevokeUser removes whitelist status.
func (s *UserStore) RevokeUser(id int64) error {
	_, err := s.DB.Exec(
		`UPDATE users SET whitelisted = 0, role = 'student' WHERE id = ?`, id,
	)
	return err
}

// UpdateUserRole updates only the role of a user.
func (s *UserStore) UpdateUserRole(id int64, role string) error {
	_, err := s.DB.Exec(`UPDATE users SET role = ? WHERE id = ?`, role, id)
	return err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (*User, error) {
	u := &User{}
	var wl int
	err := row.Scan(&u.ID, &u.GitHubLogin, &u.GitHubID, &u.Name, &u.Email,
		&u.AvatarURL, &u.Role, &wl, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.Whitelisted = wl != 0
	return u, nil
}

// ─── Cohort ──────────────────────────────────────────────────────────────────

// CohortStore handles cohort-related DB ops.
type CohortStore struct {
	DB *sql.DB
}

// CreateCohort inserts a new cohort.
func (s *CohortStore) CreateCohort(name, focus string, start, end *time.Time) (*Cohort, error) {
	var sNull, eNull sql.NullTime
	if start != nil {
		sNull = sql.NullTime{Time: *start, Valid: true}
	}
	if end != nil {
		eNull = sql.NullTime{Time: *end, Valid: true}
	}
	res, err := s.DB.Exec(
		`INSERT INTO cohorts (name, focus, start_date, end_date) VALUES (?, ?, ?, ?)`,
		name, focus, nullTimeStr(sNull), nullTimeStr(eNull),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetCohort(id)
}

// GetCohort retrieves a cohort by ID.
func (s *CohortStore) GetCohort(id int64) (*Cohort, error) {
	row := s.DB.QueryRow(
		`SELECT id, name, focus, start_date, end_date, created_at FROM cohorts WHERE id = ?`, id,
	)
	return scanCohort(row)
}

// ListCohorts returns all cohorts ordered by name.
func (s *CohortStore) ListCohorts() ([]Cohort, error) {
	rows, err := s.DB.Query(
		`SELECT id, name, focus, start_date, end_date, created_at FROM cohorts ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cs []Cohort
	for rows.Next() {
		c, err := scanCohort(rows)
		if err != nil {
			return nil, err
		}
		cs = append(cs, *c)
	}
	return cs, rows.Err()
}

// UpdateCohort modifies an existing cohort.
func (s *CohortStore) UpdateCohort(id int64, name, focus string, start, end *time.Time) error {
	var sNull, eNull sql.NullTime
	if start != nil {
		sNull = sql.NullTime{Time: *start, Valid: true}
	}
	if end != nil {
		eNull = sql.NullTime{Time: *end, Valid: true}
	}
	_, err := s.DB.Exec(
		`UPDATE cohorts SET name=?, focus=?, start_date=?, end_date=? WHERE id=?`,
		name, focus, nullTimeStr(sNull), nullTimeStr(eNull), id,
	)
	return err
}

// DeleteCohort removes a cohort (cascades to weeks/activities).
func (s *CohortStore) DeleteCohort(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM cohorts WHERE id=?`, id)
	return err
}

// EnrollUser adds a student to a cohort.
func (s *CohortStore) EnrollUser(cohortID, userID int64) error {
	_, err := s.DB.Exec(
		`INSERT OR IGNORE INTO cohort_enrollments (cohort_id, user_id) VALUES (?, ?)`,
		cohortID, userID,
	)
	return err
}

// UnenrollUser removes a student from a cohort.
func (s *CohortStore) UnenrollUser(cohortID, userID int64) error {
	_, err := s.DB.Exec(
		`DELETE FROM cohort_enrollments WHERE cohort_id=? AND user_id=?`,
		cohortID, userID,
	)
	return err
}

// ListEnrolledUsers returns all users enrolled in a cohort.
func (s *CohortStore) ListEnrolledUsers(cohortID int64) ([]User, error) {
	rows, err := s.DB.Query(`
		SELECT u.id, u.github_login, u.github_id, u.name, u.email, u.avatar_url, u.role, u.whitelisted, u.created_at
		FROM users u
		JOIN cohort_enrollments ce ON ce.user_id = u.id
		WHERE ce.cohort_id = ?
		ORDER BY u.github_login`, cohortID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func scanCohort(row scanner) (*Cohort, error) {
	c := &Cohort{}
	var sStr, eStr sql.NullString
	err := row.Scan(&c.ID, &c.Name, &c.Focus, &sStr, &eStr, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	if sStr.Valid {
		t, _ := time.Parse("2006-01-02", sStr.String)
		c.StartDate = sql.NullTime{Time: t, Valid: true}
	}
	if eStr.Valid {
		t, _ := time.Parse("2006-01-02", eStr.String)
		c.EndDate = sql.NullTime{Time: t, Valid: true}
	}
	return c, nil
}

// ─── Week ─────────────────────────────────────────────────────────────────────

// WeekStore handles week-related DB ops.
type WeekStore struct {
	DB *sql.DB
}

// CreateWeek adds a week to a cohort.
func (s *WeekStore) CreateWeek(cohortID int64, weekNumber int, title string, start *time.Time) (*Week, error) {
	var sNull sql.NullTime
	if start != nil {
		sNull = sql.NullTime{Time: *start, Valid: true}
	}
	res, err := s.DB.Exec(
		`INSERT INTO weeks (cohort_id, week_number, title, start_date) VALUES (?, ?, ?, ?)`,
		cohortID, weekNumber, title, nullTimeStr(sNull),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetWeek(id)
}

// GetWeek retrieves a week by ID.
func (s *WeekStore) GetWeek(id int64) (*Week, error) {
	row := s.DB.QueryRow(
		`SELECT id, cohort_id, week_number, title, start_date FROM weeks WHERE id=?`, id,
	)
	return scanWeek(row)
}

// ListWeeksByCohort returns all weeks for a cohort.
func (s *WeekStore) ListWeeksByCohort(cohortID int64) ([]Week, error) {
	rows, err := s.DB.Query(
		`SELECT id, cohort_id, week_number, title, start_date FROM weeks WHERE cohort_id=? ORDER BY week_number`,
		cohortID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ws []Week
	for rows.Next() {
		w, err := scanWeek(rows)
		if err != nil {
			return nil, err
		}
		ws = append(ws, *w)
	}
	return ws, rows.Err()
}

// DeleteWeek removes a week.
func (s *WeekStore) DeleteWeek(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM weeks WHERE id=?`, id)
	return err
}

func scanWeek(row scanner) (*Week, error) {
	w := &Week{}
	var sStr sql.NullString
	err := row.Scan(&w.ID, &w.CohortID, &w.WeekNumber, &w.Title, &sStr)
	if err != nil {
		return nil, err
	}
	if sStr.Valid {
		t, _ := time.Parse("2006-01-02", sStr.String)
		w.StartDate = sql.NullTime{Time: t, Valid: true}
	}
	return w, nil
}

// ─── Activity ─────────────────────────────────────────────────────────────────

// ActivityStore handles activity DB ops.
type ActivityStore struct {
	DB *sql.DB
}

// CreateActivity inserts a new activity.
func (s *ActivityStore) CreateActivity(a Activity) (*Activity, error) {
	res, err := s.DB.Exec(`
		INSERT INTO activities (week_id, title, description, source_url, assigned_date, due_date, is_group, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.WeekID, a.Title, a.Description, a.SourceURL,
		nullTimeStr(a.AssignedDate), nullTimeStr(a.DueDate),
		boolInt(a.IsGroup), a.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetActivity(id)
}

// GetActivity retrieves an activity by ID.
func (s *ActivityStore) GetActivity(id int64) (*Activity, error) {
	row := s.DB.QueryRow(`
		SELECT id, week_id, title, description, source_url, assigned_date, due_date, is_group, created_by, created_at
		FROM activities WHERE id=?`, id,
	)
	return scanActivity(row)
}

// ListActivitiesByWeek returns activities for a given week.
func (s *ActivityStore) ListActivitiesByWeek(weekID int64) ([]Activity, error) {
	rows, err := s.DB.Query(`
		SELECT id, week_id, title, description, source_url, assigned_date, due_date, is_group, created_by, created_at
		FROM activities WHERE week_id=? ORDER BY assigned_date, title`, weekID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivities(rows)
}

// UpdateActivity modifies an existing activity.
func (s *ActivityStore) UpdateActivity(a Activity) error {
	_, err := s.DB.Exec(`
		UPDATE activities
		SET title=?, description=?, source_url=?, assigned_date=?, due_date=?, is_group=?
		WHERE id=?`,
		a.Title, a.Description, a.SourceURL,
		nullTimeStr(a.AssignedDate), nullTimeStr(a.DueDate),
		boolInt(a.IsGroup), a.ID,
	)
	return err
}

// DeleteActivity removes an activity.
func (s *ActivityStore) DeleteActivity(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM activities WHERE id=?`, id)
	return err
}

func scanActivity(row scanner) (*Activity, error) {
	a := &Activity{}
	var adStr, ddStr sql.NullString
	var isGroup int
	err := row.Scan(&a.ID, &a.WeekID, &a.Title, &a.Description, &a.SourceURL,
		&adStr, &ddStr, &isGroup, &a.CreatedBy, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	a.IsGroup = isGroup != 0
	if adStr.Valid {
		t, _ := time.Parse("2006-01-02", adStr.String)
		a.AssignedDate = sql.NullTime{Time: t, Valid: true}
	}
	if ddStr.Valid {
		t, _ := time.Parse("2006-01-02", ddStr.String)
		a.DueDate = sql.NullTime{Time: t, Valid: true}
	}
	return a, nil
}

func scanActivities(rows *sql.Rows) ([]Activity, error) {
	var as []Activity
	for rows.Next() {
		a := &Activity{}
		var adStr, ddStr sql.NullString
		var isGroup int
		err := rows.Scan(&a.ID, &a.WeekID, &a.Title, &a.Description, &a.SourceURL,
			&adStr, &ddStr, &isGroup, &a.CreatedBy, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		a.IsGroup = isGroup != 0
		if adStr.Valid {
			t, _ := time.Parse("2006-01-02", adStr.String)
			a.AssignedDate = sql.NullTime{Time: t, Valid: true}
		}
		if ddStr.Valid {
			t, _ := time.Parse("2006-01-02", ddStr.String)
			a.DueDate = sql.NullTime{Time: t, Valid: true}
		}
		as = append(as, *a)
	}
	return as, rows.Err()
}

// ─── Submission ───────────────────────────────────────────────────────────────

// SubmissionStore handles submission DB ops.
type SubmissionStore struct {
	DB *sql.DB
}

// CreateSubmission inserts a new submission.
func (s *SubmissionStore) CreateSubmission(sub Submission) (*Submission, error) {
	res, err := s.DB.Exec(`
		INSERT INTO submissions (activity_id, user_id, group_id, repo_url, notes)
		VALUES (?, ?, ?, ?, ?)`,
		sub.ActivityID, sub.UserID, sub.GroupID, sub.RepoURL, sub.Notes,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetSubmission(id)
}

// GetSubmission retrieves a submission by ID.
func (s *SubmissionStore) GetSubmission(id int64) (*Submission, error) {
	row := s.DB.QueryRow(`
		SELECT id, activity_id, user_id, group_id, repo_url, notes, submitted_at, updated_at
		FROM submissions WHERE id=?`, id,
	)
	return scanSubmission(row)
}

// UpdateSubmission changes repo_url and notes.
func (s *SubmissionStore) UpdateSubmission(id int64, repoURL, notes string) error {
	_, err := s.DB.Exec(`
		UPDATE submissions SET repo_url=?, notes=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		repoURL, notes, id,
	)
	return err
}

// DeleteSubmission removes a submission.
func (s *SubmissionStore) DeleteSubmission(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM submissions WHERE id=?`, id)
	return err
}

// ListSubmissionsByActivity returns all submissions for an activity with user info.
func (s *SubmissionStore) ListSubmissionsByActivity(activityID int64) ([]Submission, error) {
	rows, err := s.DB.Query(`
		SELECT s.id, s.activity_id, s.user_id, s.group_id, s.repo_url, s.notes, s.submitted_at, s.updated_at,
		       u.id, u.github_login, u.github_id, u.name, u.email, u.avatar_url, u.role, u.whitelisted, u.created_at
		FROM submissions s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.activity_id=?
		ORDER BY s.submitted_at`, activityID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []Submission
	for rows.Next() {
		sub, u, err := scanSubmissionWithUser(rows)
		if err != nil {
			return nil, err
		}
		sub.User = u
		subs = append(subs, *sub)
	}
	return subs, rows.Err()
}

// GetSubmissionByUserActivity returns the submission for a user + activity pair (if any).
func (s *SubmissionStore) GetSubmissionByUserActivity(userID, activityID int64) (*Submission, error) {
	row := s.DB.QueryRow(`
		SELECT id, activity_id, user_id, group_id, repo_url, notes, submitted_at, updated_at
		FROM submissions WHERE user_id=? AND activity_id=?`, userID, activityID,
	)
	return scanSubmission(row)
}

func scanSubmission(row scanner) (*Submission, error) {
	s := &Submission{}
	err := row.Scan(&s.ID, &s.ActivityID, &s.UserID, &s.GroupID,
		&s.RepoURL, &s.Notes, &s.SubmittedAt, &s.UpdatedAt)
	return s, err
}

type submissionWithUserScanner interface {
	Scan(dest ...any) error
}

func scanSubmissionWithUser(rows submissionWithUserScanner) (*Submission, *User, error) {
	sub := &Submission{}
	u := &User{}
	var wl int
	err := rows.Scan(
		&sub.ID, &sub.ActivityID, &sub.UserID, &sub.GroupID,
		&sub.RepoURL, &sub.Notes, &sub.SubmittedAt, &sub.UpdatedAt,
		&u.ID, &u.GitHubLogin, &u.GitHubID, &u.Name, &u.Email,
		&u.AvatarURL, &u.Role, &wl, &u.CreatedAt,
	)
	if err != nil {
		return nil, nil, err
	}
	u.Whitelisted = wl != 0
	return sub, u, nil
}

// ─── Group ────────────────────────────────────────────────────────────────────

// GroupStore handles group DB ops.
type GroupStore struct {
	DB *sql.DB
}

// CreateGroup creates a new group for an activity.
func (s *GroupStore) CreateGroup(activityID int64, name string) (*Group, error) {
	res, err := s.DB.Exec(
		`INSERT INTO groups (activity_id, name) VALUES (?, ?)`,
		activityID, name,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetGroup(id)
}

// GetGroup retrieves a group with its members.
func (s *GroupStore) GetGroup(id int64) (*Group, error) {
	row := s.DB.QueryRow(
		`SELECT id, activity_id, name, created_at FROM groups WHERE id=?`, id,
	)
	g := &Group{}
	err := row.Scan(&g.ID, &g.ActivityID, &g.Name, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	members, err := s.ListGroupMembers(id)
	if err != nil {
		return nil, err
	}
	g.Members = members
	return g, nil
}

// ListGroupsByActivity lists all groups for an activity.
func (s *GroupStore) ListGroupsByActivity(activityID int64) ([]Group, error) {
	rows, err := s.DB.Query(
		`SELECT id, activity_id, name, created_at FROM groups WHERE activity_id=? ORDER BY name`,
		activityID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var gs []Group
	for rows.Next() {
		g := &Group{}
		err := rows.Scan(&g.ID, &g.ActivityID, &g.Name, &g.CreatedAt)
		if err != nil {
			return nil, err
		}
		members, _ := s.ListGroupMembers(g.ID)
		g.Members = members
		gs = append(gs, *g)
	}
	return gs, rows.Err()
}

// AddMember adds a user to a group.
func (s *GroupStore) AddMember(groupID, userID int64) error {
	_, err := s.DB.Exec(
		`INSERT OR IGNORE INTO group_members (group_id, user_id) VALUES (?, ?)`,
		groupID, userID,
	)
	return err
}

// RemoveMember removes a user from a group.
func (s *GroupStore) RemoveMember(groupID, userID int64) error {
	_, err := s.DB.Exec(
		`DELETE FROM group_members WHERE group_id=? AND user_id=?`,
		groupID, userID,
	)
	return err
}

// ListGroupMembers returns all members of a group.
func (s *GroupStore) ListGroupMembers(groupID int64) ([]User, error) {
	rows, err := s.DB.Query(`
		SELECT u.id, u.github_login, u.github_id, u.name, u.email, u.avatar_url, u.role, u.whitelisted, u.created_at
		FROM users u
		JOIN group_members gm ON gm.user_id = u.id
		WHERE gm.group_id=?
		ORDER BY u.github_login`, groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var us []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		us = append(us, *u)
	}
	return us, rows.Err()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func nullTimeStr(t sql.NullTime) interface{} {
	if !t.Valid {
		return nil
	}
	return t.Time.Format("2006-01-02")
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
