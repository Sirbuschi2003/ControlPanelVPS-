package executor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/agent/internal/version"
)

const (
	releaseBase    = "https://github.com/Sirbuschi2003/ControlPanelVPS-/releases/download/latest"
	githubReleaseAPI = "https://api.github.com/repos/Sirbuschi2003/ControlPanelVPS-/releases/tags/latest"
)

type UpdateResult struct {
	PreviousCommit string    `json:"previous_commit"`
	NewCommit      string    `json:"new_commit"`
	ChangedFiles   int       `json:"changed_files"`
	Output         string    `json:"output"`
	Duration       string    `json:"duration"`
	RestartedAt    time.Time `json:"restarted_at"`
}

type githubRelease struct {
	Body string `json:"body"`
}

// GetVersion returns the embedded build-time version info.
func GetVersion() (commit, date, branch string) {
	return version.Commit, version.Date, "master"
}

// fetchLatestCommit queries the GitHub Releases API and extracts the commit SHA from the release body.
func fetchLatestCommit() (string, error) {
	req, err := http.NewRequest(http.MethodGet, githubReleaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ControlPanelVPS-Agent")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("decode release: %w", err)
	}

	for _, line := range strings.Split(rel.Body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "commit:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "commit:")), nil
		}
	}
	return "", nil
}

// CheckUpdates returns true if the GitHub release commit differs from the embedded build commit.
func CheckUpdates() (available bool, latestCommit string, err error) {
	latestCommit, err = fetchLatestCommit()
	if err != nil {
		return false, "", err
	}
	if latestCommit == "" || version.Commit == "dev" {
		return false, latestCommit, nil
	}
	available = latestCommit != version.Commit
	return
}

// RunUpdate downloads pre-built release artifacts, replaces binaries, and restarts services.
func RunUpdate() (*UpdateResult, error) {
	start := time.Now()
	dir := installDir()
	var sb strings.Builder

	log := func(msg string) {
		sb.WriteString("[" + time.Now().Format("15:04:05") + "] " + msg + "\n")
	}

	prevCommit := shortCommit(version.Commit)
	log("Aktueller Commit: " + prevCommit)

	latestCommit, err := fetchLatestCommit()
	if err != nil {
		log("WARN: Konnte neuesten Commit nicht abrufen: " + err.Error())
	}
	newCommit := shortCommit(latestCommit)
	log("Neuer Commit: " + newCommit)

	// Master binary
	log("Lade Master-API Binary herunter...")
	if err := downloadBinary(releaseBase+"/master", filepath.Join(dir, "bin", "master")); err != nil {
		return nil, fmt.Errorf("download master: %w", err)
	}
	log("Master-API heruntergeladen.")

	// Agent binary
	log("Lade Agent Binary herunter...")
	if err := downloadBinary(releaseBase+"/agent", filepath.Join(dir, "bin", "agent")); err != nil {
		return nil, fmt.Errorf("download agent: %w", err)
	}
	log("Agent heruntergeladen.")

	// Frontend
	log("Lade Frontend herunter...")
	standaloneDir := filepath.Join(dir, "frontend-standalone")
	if err := os.MkdirAll(standaloneDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir frontend-standalone: %w", err)
	}
	if err := downloadAndExtractTarGz(releaseBase+"/frontend.tar.gz", standaloneDir); err != nil {
		return nil, fmt.Errorf("download frontend: %w", err)
	}
	log("Frontend heruntergeladen und entpackt.")

	// Restart services
	log("Starte Dienste neu...")
	for _, svc := range []string{"cpanel-master", "cpanel-agent", "cpanel-frontend"} {
		if out, err := runCmdOutput("systemctl", "restart", svc); err != nil {
			log("WARN: restart " + svc + ": " + out)
		} else {
			log(svc + " neu gestartet.")
		}
	}

	return &UpdateResult{
		PreviousCommit: prevCommit,
		NewCommit:      newCommit,
		ChangedFiles:   0,
		Output:         sb.String(),
		Duration:       time.Since(start).Round(time.Second).String(),
		RestartedAt:    time.Now(),
	}, nil
}

// downloadBinary downloads url to dest using a temp file + remove + rename
// to avoid ETXTBSY when replacing a running binary.
func downloadBinary(url, dest string) error {
	tmpPath := dest + ".new"

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	f.Close()

	// Remove running binary first — kernel keeps fd open in memory, so the
	// process stays alive while the new binary is placed at the same path.
	_ = os.Remove(dest)
	return os.Rename(tmpPath, dest)
}

func downloadAndExtractTarGz(url, destDir string) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	cmd := exec.Command("tar", "-xz", "-C", destDir)
	cmd.Stdin = resp.Body
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar: %w\n%s", err, out)
	}
	return nil
}

func shortCommit(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}

func installDir() string {
	if d := os.Getenv("INSTALL_DIR"); d != "" {
		return d
	}
	return "/opt/controlpanel"
}

func runCmdOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
