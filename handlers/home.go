package handlers

import "net/http"

// Home is the landing page.
func (a *App) Home(w http.ResponseWriter, r *http.Request) {
	cohorts, _ := a.Cohorts.ListCohorts()
	u := currentUser(r)
	renderTemplate(w, "index.html", map[string]interface{}{
		"Cohorts":     cohorts,
		"CurrentUser": u,
		"IsLocalAdmin": isLocalAdmin(r),
	})
}
