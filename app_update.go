package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// UpdateInfo is returned to the frontend by CheckForUpdate.
type UpdateInfo struct {
	Current         string `json:"current"`         // version of this build
	Latest          string `json:"latest"`          // newest release tag on GitHub
	UpdateAvailable bool   `json:"updateAvailable"` // true if Latest > Current
	URL             string `json:"url"`             // release page to download from
	SelfUpdate      bool   `json:"selfUpdate"`      // true if the release carries a binary this build can swap to
}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type githubRelease struct {
	TagName string         `json:"tag_name"`
	HTMLURL string         `json:"html_url"`
	Assets  []releaseAsset `json:"assets"`
}

// updateAssetName is the release asset carrying a raw binary for this
// platform (built by .github/workflows/release.yml). macOS ships a single
// universal binary, so the name is arch-independent there.
func updateAssetName() string {
	switch runtime.GOOS {
	case "windows":
		return "InkLab-update-windows-" + runtime.GOARCH + ".exe"
	case "darwin":
		return "InkLab-update-darwin-universal"
	default:
		return "InkLab-update-" + runtime.GOOS + "-" + runtime.GOARCH
	}
}

func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", Repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "InkLab")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func (r *githubRelease) asset(name string) *releaseAsset {
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			return &r.Assets[i]
		}
	}
	return nil
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

	release, err := fetchLatestRelease()
	if err != nil {
		return info, err
	}

	info.Latest = release.TagName
	info.URL = release.HTMLURL
	info.UpdateAvailable = isNewerVersion(Version, release.TagName)
	info.SelfUpdate = info.UpdateAvailable && release.asset(updateAssetName()) != nil
	return info, nil
}

// ApplyUpdate downloads the latest release binary for this platform, verifies
// its checksum, and swaps it in place of the running executable. The running
// process keeps executing the old code; call Restart to pick up the new
// binary. Emits "update:progress" {downloaded, total} while downloading.
func (a *App) ApplyUpdate() error {
	release, err := fetchLatestRelease()
	if err != nil {
		return err
	}
	if !isNewerVersion(Version, release.TagName) {
		return fmt.Errorf("no newer release available")
	}

	asset := release.asset(updateAssetName())
	if asset == nil {
		return fmt.Errorf("release %s has no update binary for this platform", release.TagName)
	}

	wantHash, err := fetchChecksum(release, asset.Name)
	if err != nil {
		return err
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	newPath := exePath + ".new"

	if err := a.downloadUpdate(asset, newPath); err != nil {
		os.Remove(newPath)
		return err
	}

	gotHash, err := fileSHA256(newPath)
	if err != nil {
		os.Remove(newPath)
		return err
	}
	if gotHash != wantHash {
		os.Remove(newPath)
		return fmt.Errorf("checksum mismatch: got %s, want %s", gotHash, wantHash)
	}

	if err := os.Chmod(newPath, 0o755); err != nil {
		os.Remove(newPath)
		return err
	}

	return swapExecutable(exePath, newPath)
}

// Restart launches the (freshly swapped) executable and quits this instance.
func (a *App) Restart() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exePath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	wailsruntime.Quit(a.ctx)
	return nil
}

// cleanupOldUpdate removes leftovers of a previous self-update. On Windows the
// old binary can't be deleted while it is still running, so the swap leaves it
// behind as .old and we collect it on the next start.
func cleanupOldUpdate() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	os.Remove(exePath + ".old")
	os.Remove(exePath + ".new")
}

// fetchChecksum downloads "<asset>.sha256" from the release and returns the
// hex digest (first whitespace-separated field, sha256sum format).
func fetchChecksum(release *githubRelease, assetName string) (string, error) {
	sumAsset := release.asset(assetName + ".sha256")
	if sumAsset == nil {
		return "", fmt.Errorf("release has no checksum for %s", assetName)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(sumAsset.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(body))
	if len(fields) == 0 || len(fields[0]) != 64 {
		return "", fmt.Errorf("malformed checksum file for %s", assetName)
	}
	return strings.ToLower(fields[0]), nil
}

func (a *App) downloadUpdate(asset *releaseAsset, dest string) error {
	client := &http.Client{Timeout: 15 * time.Minute}
	resp, err := client.Get(asset.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update download returned status %d", resp.StatusCode)
	}

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	var downloaded int64
	lastEmit := time.Now()
	buf := make([]byte, 256*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if time.Since(lastEmit) > 200*time.Millisecond {
				lastEmit = time.Now()
				wailsruntime.EventsEmit(a.ctx, "update:progress", map[string]interface{}{
					"downloaded": downloaded,
					"total":      asset.Size,
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	wailsruntime.EventsEmit(a.ctx, "update:progress", map[string]interface{}{
		"downloaded": downloaded,
		"total":      asset.Size,
	})
	return out.Close()
}

// swapExecutable puts newPath in place of exePath. On Windows a running
// executable can't be overwritten but can be renamed, so the current binary
// is parked as .old first (removed on next start); if the final rename fails
// the old binary is rolled back. On Unix rename() replaces the path atomically
// while the running process keeps its open inode.
func swapExecutable(exePath, newPath string) error {
	if runtime.GOOS == "windows" {
		oldPath := exePath + ".old"
		os.Remove(oldPath)
		if err := os.Rename(exePath, oldPath); err != nil {
			os.Remove(newPath)
			return err
		}
		if err := os.Rename(newPath, exePath); err != nil {
			os.Rename(oldPath, exePath)
			os.Remove(newPath)
			return err
		}
		return nil
	}
	return os.Rename(newPath, exePath)
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
