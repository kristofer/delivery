package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kristofer/delivery/models"
)

// ─── Cohorts ──────────────────────────────────────────────────────────────────

// CohortList lists all cohorts (instructor/admin view).
func (a *App) CohortList(w http.ResponseWriter, r *http.Request) {
	cohorts, _ := a.Cohorts.ListCohorts()
	renderTemplate(w, "cohorts.html", map[string]interface{}{
		"Cohorts":     cohorts,
		"CurrentUser": currentUser(r),
		"IsLocalAdmin": isLocalAdmin(r),
	})
}

// CohortNew shows the create-cohort form.
func (a *App) CohortNew(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "cohort_form.html", map[string]interface{}{
		"CurrentUser": currentUser(r),
		"IsLocalAdmin": isLocalAdmin(r),
	})
}

// CohortCreate handles form submission for a new cohort.
func (a *App) CohortCreate(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	focus := r.FormValue("focus")
	start := parseDate(r.FormValue("start_date"))
	end := parseDate(r.FormValue("end_date"))
	_, err := a.Cohorts.CreateCohort(name, focus, start, end)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cohorts", http.StatusFound)
}

// CohortShow shows a cohort with its weeks and activities.
func (a *App) CohortShow(w http.ResponseWriter, r *http.Request) {
	id := parseIDParam(r, "id")
	cohort, err := a.Cohorts.GetCohort(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	weeks, _ := a.Weeks.ListWeeksByCohort(id)
	enrolled, _ := a.Cohorts.ListEnrolledUsers(id)

	type weekWithActs struct {
		Week       interface{}
		Activities interface{}
	}
	type weekData struct {
		ID         int64
		WeekNumber int
		Title      string
		StartDate  interface{}
		Activities interface{}
	}
	var weekItems []weekData
	for _, w := range weeks {
		acts, _ := a.Acts.ListActivitiesByWeek(w.ID)
		weekItems = append(weekItems, weekData{
			ID:         w.ID,
			WeekNumber: w.WeekNumber,
			Title:      w.Title,
			StartDate:  w.StartDate,
			Activities: acts,
		})
	}

	renderTemplate(w, "cohort_detail.html", map[string]interface{}{
		"Cohort":      cohort,
		"Weeks":       weekItems,
		"Enrolled":    enrolled,
		"CurrentUser": currentUser(r),
		"IsLocalAdmin": isLocalAdmin(r),
	})
}

// CohortEdit shows the edit form.
func (a *App) CohortEdit(w http.ResponseWriter, r *http.Request) {
	id := parseIDParam(r, "id")
	cohort, err := a.Cohorts.GetCohort(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "cohort_form.html", map[string]interface{}{
		"Cohort":      cohort,
		"CurrentUser": currentUser(r),
		"IsLocalAdmin": isLocalAdmin(r),
	})
}

// CohortUpdate handles the edit form POST.
func (a *App) CohortUpdate(w http.ResponseWriter, r *http.Request) {
	id := parseIDParam(r, "id")
	name := r.FormValue("name")
	focus := r.FormValue("focus")
	start := parseDate(r.FormValue("start_date"))
	end := parseDate(r.FormValue("end_date"))
	if err := a.Cohorts.UpdateCohort(id, name, focus, start, end); err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(id, 10), http.StatusFound)
}

// CohortDelete removes a cohort.
func (a *App) CohortDelete(w http.ResponseWriter, r *http.Request) {
	id := parseIDParam(r, "id")
	_ = a.Cohorts.DeleteCohort(id)
	http.Redirect(w, r, "/cohorts", http.StatusFound)
}

// CohortEnroll adds a student to a cohort.
func (a *App) CohortEnroll(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	userIDStr := r.FormValue("user_id")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)
	_ = a.Cohorts.EnrollUser(cohortID, userID)
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10), http.StatusFound)
}

// CohortUnenroll removes a student from a cohort.
func (a *App) CohortUnenroll(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	userID := parseIDParam(r, "uid")
	_ = a.Cohorts.UnenrollUser(cohortID, userID)
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10), http.StatusFound)
}

// ─── Weeks ────────────────────────────────────────────────────────────────────

// WeekCreate adds a week to a cohort.
func (a *App) WeekCreate(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	numStr := r.FormValue("week_number")
	num, _ := strconv.Atoi(numStr)
	title := r.FormValue("title")
	start := parseDate(r.FormValue("start_date"))
	_, err := a.Weeks.CreateWeek(cohortID, num, title, start)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10), http.StatusFound)
}

// WeekDelete removes a week.
func (a *App) WeekDelete(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	weekID := parseIDParam(r, "wid")
	_ = a.Weeks.DeleteWeek(weekID)
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10), http.StatusFound)
}

// ─── Activities ───────────────────────────────────────────────────────────────

// ActivityNew shows the activity creation form.
func (a *App) ActivityNew(w http.ResponseWriter, r *http.Request) {
	weekID := parseIDParam(r, "wid")
	cohortID := parseIDParam(r, "id")
	renderTemplate(w, "activity_form.html", map[string]interface{}{
		"WeekID":      weekID,
		"CohortID":    cohortID,
		"CurrentUser": currentUser(r),
		"IsLocalAdmin": isLocalAdmin(r),
	})
}

// ActivityCreate handles the new-activity form POST.
func (a *App) ActivityCreate(w http.ResponseWriter, r *http.Request) {
	weekID := parseIDParam(r, "wid")
	cohortID := parseIDParam(r, "id")

	act := buildActivity(r, weekID)
	_, err := a.Acts.CreateActivity(act)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10), http.StatusFound)
}

// ActivityShow shows an activity with its submissions.
func (a *App) ActivityShow(w http.ResponseWriter, r *http.Request) {
	actID := parseIDParam(r, "aid")
	act, err := a.Acts.GetActivity(actID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	subs, _ := a.Subs.ListSubmissionsByActivity(actID)
	groups, _ := a.Groups.ListGroupsByActivity(actID)
	u := currentUser(r)

	var mySubmission interface{}
	if u != nil {
		s, _ := a.Subs.GetSubmissionByUserActivity(u.ID, actID)
		if s != nil {
			mySubmission = s
		}
	}

	renderTemplate(w, "activity_detail.html", map[string]interface{}{
		"Activity":     act,
		"Submissions":  subs,
		"Groups":       groups,
		"MySubmission": mySubmission,
		"CurrentUser":  u,
		"IsLocalAdmin": isLocalAdmin(r),
	})
}

// ActivityEdit shows the edit form.
func (a *App) ActivityEdit(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	actID := parseIDParam(r, "aid")
	act, err := a.Acts.GetActivity(actID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "activity_form.html", map[string]interface{}{
		"Activity":    act,
		"CohortID":    cohortID,
		"CurrentUser": currentUser(r),
		"IsLocalAdmin": isLocalAdmin(r),
	})
}

// ActivityUpdate handles the edit form POST.
func (a *App) ActivityUpdate(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	actID := parseIDParam(r, "aid")
	act, err := a.Acts.GetActivity(actID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	updated := buildActivity(r, act.WeekID)
	updated.ID = actID
	if err := a.Acts.UpdateActivity(updated); err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10)+"/activities/"+strconv.FormatInt(actID, 10), http.StatusFound)
}

// ActivityDelete removes an activity.
func (a *App) ActivityDelete(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	actID := parseIDParam(r, "aid")
	_ = a.Acts.DeleteActivity(actID)
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10), http.StatusFound)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func parseIDParam(r *http.Request, param string) int64 {
	s := chi.URLParam(r, param)
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func buildActivity(r *http.Request, weekID int64) models.Activity {
	isGroup := r.FormValue("is_group") == "1"
	a := models.Activity{
		WeekID:      weekID,
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		SourceURL:   r.FormValue("source_url"),
		IsGroup:     isGroup,
	}
	if ad := parseDate(r.FormValue("assigned_date")); ad != nil {
		a.AssignedDate = sql.NullTime{Time: *ad, Valid: true}
	}
	if dd := parseDate(r.FormValue("due_date")); dd != nil {
		a.DueDate = sql.NullTime{Time: *dd, Valid: true}
	}
	return a
}
