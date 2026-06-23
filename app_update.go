package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// UpdateInfo is returned to the frontend by CheckForUpdate.
type UpdateInfo struct {
	Current         string `json:"current"`         // version of this build
	Latest          string `json:"latest"`          // newest release tag on GitHub
	UpdateAvailable bool   `json:"updateAvailable"` // true if Latest > Current
	URL             string `json:"url"`             // release page to download from
}

// CheckForUpdate asks GitHub whether a newer release exists for the repository
// this build was produced from. Version and Repo are injected via -ldflags at
// release time; for local/dev builds they are unset, so this is a no-op
// (UpdateAvailable stays false and the UI shows nothing).
func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	info := &UpdateInfo{Current: Version}

	if Version == "dev" || Version == "" || Repo == "" {
		return info, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", Repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "InkLab")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("github returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return info, err
	}

	info.Latest = release.TagName
	info.URL = release.HTMLURL
	info.UpdateAvailable = isNewerVersion(Version, release.TagName)
	return info, nil
}

// isNewerVersion reports whether latest is a higher semantic version than
// current. A leading "v" and any pre-release/build suffix are ignored.
func isNewerVersion(current, latest string) bool {
	c := parseVersion(current)
	l := parseVersion(latest)
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// parseVersion turns "v1.2.3" / "1.2.3-rc1" into [1,2,3].
func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var out [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(strings.TrimSpace(part))
	}
	return out
}
