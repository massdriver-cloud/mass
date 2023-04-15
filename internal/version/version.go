package version

import (
	"net/http"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	LatestReleaseURL = "https://github.com/massdriver-cloud/mass/releases/latest"
)

// var needs to be used instead of const as ldflags is used to fill this
// information in the release process
var (
	version = "unknown" // this will be the release tag
)

// MassVersion returns the current version of the github.com/massdriver-cloud/mass.
func MassVersion() string {
	return version
}

func CheckForNewerVersionAvailable() (bool, string, error) {
	resp, err := http.Get(LatestReleaseURL) //nolint:noctx
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()
	// Github will redirect releases/latest to the appropriate releases/tag/X.X.X
	redirectURL := resp.Request.URL.String()
	parts := strings.Split(redirectURL, "/")
	latestVersion := parts[len(parts)-1]
	if semver.Compare(version, latestVersion) < 0 {
		return true, latestVersion, nil
	}
	return false, latestVersion, nil
}

func SetVersion(v string) {
	version = v
}
