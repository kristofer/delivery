package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kristofer/delivery/models"
)

// SubmissionCreate handles a student submitting a repo URL.
func (a *App) SubmissionCreate(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	actID := parseIDParam(r, "aid")
	u := currentUser(r)
	if u == nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	repoURL := r.FormValue("repo_url")
	notes := r.FormValue("notes")

	sub := models.Submission{
		ActivityID: actID,
		RepoURL:    repoURL,
		Notes:      notes,
	}

	// Check if it's a group submission
	groupIDStr := r.FormValue("group_id")
	if groupIDStr != "" {
		gid, _ := strconv.ParseInt(groupIDStr, 10, 64)
		sub.GroupID.Int64 = gid
		sub.GroupID.Valid = true
	} else {
		sub.UserID.Int64 = u.ID
		sub.UserID.Valid = true
	}

	_, err := a.Subs.CreateSubmission(sub)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10)+"/activities/"+strconv.FormatInt(actID, 10), http.StatusFound)
}

// SubmissionUpdate changes the repo URL of an existing submission.
func (a *App) SubmissionUpdate(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	actID := parseIDParam(r, "aid")
	subID := parseIDParam(r, "sid")
	u := currentUser(r)

	// Verify ownership (student can only edit their own)
	if u != nil && u.Role == models.RoleStudent {
		existing, err := a.Subs.GetSubmission(subID)
		if err != nil || !existing.UserID.Valid || existing.UserID.Int64 != u.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	repoURL := r.FormValue("repo_url")
	notes := r.FormValue("notes")
	if err := a.Subs.UpdateSubmission(subID, repoURL, notes); err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10)+"/activities/"+strconv.FormatInt(actID, 10), http.StatusFound)
}

// SubmissionDelete removes a submission.
func (a *App) SubmissionDelete(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	actID := parseIDParam(r, "aid")
	subID := parseIDParam(r, "sid")
	u := currentUser(r)

	if u != nil && u.Role == models.RoleStudent {
		existing, err := a.Subs.GetSubmission(subID)
		if err != nil || !existing.UserID.Valid || existing.UserID.Int64 != u.ID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	_ = a.Subs.DeleteSubmission(subID)
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10)+"/activities/"+strconv.FormatInt(actID, 10), http.StatusFound)
}

// GroupCreate creates a group for an activity.
func (a *App) GroupCreate(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	actID := parseIDParam(r, "aid")
	name := r.FormValue("name")
	_, err := a.Groups.CreateGroup(actID, name)
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10)+"/activities/"+strconv.FormatInt(actID, 10), http.StatusFound)
}

// GroupAddMember adds a user to a group.
func (a *App) GroupAddMember(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	actID := parseIDParam(r, "aid")
	groupID := parseIDParam(r, "gid")
	userIDStr := r.FormValue("user_id")
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)
	_ = a.Groups.AddMember(groupID, userID)
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10)+"/activities/"+strconv.FormatInt(actID, 10), http.StatusFound)
}

// GroupRemoveMember removes a user from a group.
func (a *App) GroupRemoveMember(w http.ResponseWriter, r *http.Request) {
	cohortID := parseIDParam(r, "id")
	actID := parseIDParam(r, "aid")
	groupID := parseIDParam(r, "gid")
	userID := parseIDParam(r, "uid")
	_ = a.Groups.RemoveMember(groupID, userID)
	http.Redirect(w, r, "/cohorts/"+strconv.FormatInt(cohortID, 10)+"/activities/"+strconv.FormatInt(actID, 10), http.StatusFound)
}

// unused import guard
var _ = chi.URLParam
