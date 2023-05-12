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
	gitSHA  = "unknown" // sha1 from git, output of $(git rev-parse HEAD)
)

// MassVersion returns the current version of the github.com/massdriver-cloud/mass.
func MassVersion() string {
	return version
}

func MassGitSHA() string {
	return gitSHA
}

func SetVersion(setVersion string) {
	version = setVersion
}

var GetLatestVersion = func() (string, error) {
	resp, err := http.Get(LatestReleaseURL) //nolint:noctx
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// Github will redirect releases/latest to the appropriate releases/tag/X.X.X
	redirectURL := resp.Request.URL.String()
	parts := strings.Split(redirectURL, "/")
	latestVersion := parts[len(parts)-1]
	return latestVersion, nil
}

func CheckForNewerVersionAvailable() (bool, string, error) {
	currentVersion := version
	latestVersion, err := GetLatestVersion()
	if err != nil {
		return false, "", err
	}
	// semver requires a "v" prefix (v1.0.0 not 1.0.0), so add prefix if missing
	if !strings.HasPrefix(currentVersion, "v") {
		currentVersion = "v" + currentVersion
	}
	if !strings.HasPrefix(latestVersion, "v") {
		latestVersion = "v" + latestVersion
	}

	if semver.Compare(currentVersion, latestVersion) < 0 {
		return true, latestVersion, nil
	}
	return false, latestVersion, nil
}

// func GetLatestVersion() (string, error) {
// 	resp, err := http.Get(LatestReleaseURL) //nolint:noctx
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()
// 	// Github will redirect releases/latest to the appropriate releases/tag/X.X.X
// 	redirectURL := resp.Request.URL.String()
// 	parts := strings.Split(redirectURL, "/")
// 	latestVersion := parts[len(parts)-1]
// 	return latestVersion, nil
// }
