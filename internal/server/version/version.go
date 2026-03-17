// Package version provides an HTTP handler for reporting the current CLI version.
package version

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/massdriver-cloud/mass/internal/version"
)

// Version holds version comparison information for the CLI.
type Version struct {
	IsLatest       bool   `json:"isLatest"`
	LatestVersion  string `json:"latestVersion"`
	CurrentVersion string `json:"currentVersion"`
}

// Latest handles an HTTP request and responds with the current and latest CLI version info.
func Latest(w http.ResponseWriter, _ *http.Request) {
	latest, err := version.GetLatestVersion()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	isLatest, _ := version.CheckForNewerVersionAvailable(latest)

	v := Version{
		IsLatest:       !isLatest,
		LatestVersion:  latest,
		CurrentVersion: version.MassVersion(),
	}
	out, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(out); err != nil {
		slog.Error("Error writing version response", "error", err.Error())
	}
}
