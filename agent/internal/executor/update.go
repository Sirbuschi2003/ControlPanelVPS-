package executor

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const releaseBase = "https://github.com/Sirbuschi2003/ControlPanelVPS-/releases/download/latest"

type UpdateResult struct {
	PreviousCommit string    `json:"previous_commit"`
	NewCommit      string    `json:"new_commit"`
	ChangedFiles   int       `json:"changed_files"`
	Output         string    `json:"output"`
	Duration       string    `json:"duration"`
	RestartedAt    time.Time `json:"restarted_at"`
}

// GetVersion returns the current git commit hash and date.
func GetVersion() (commit, date, branch string) {
	commit = runCmd("git", "-C", installDir(), "rev-parse", "--short", "HEAD")
	date = runCmd("git", "-C", installDir(), "log", "-1", "--format=%ci")
	branch = runCmd("git", "-C", installDir(), "rev-parse", "--abbrev-ref", "HEAD")
	return
}

// CheckUpdates returns true if upstream has new commits.
func CheckUpdates() (available bool, latestCommit string, err error) {
	dir := installDir()
	if err2 := runCmdErr("git", "-C", dir, "fetch", "origin"); err2 != nil {
		return false, "", fmt.Errorf("git fetch: %w", err2)
	}
	local := strings.TrimSpace(runCmd("git", "-C", dir, "rev-parse", "HEAD"))
	remote := strings.TrimSpace(runCmd("git", "-C", dir, "rev-parse", "@{u}"))
	latestCommit = remote
	available = local != remote && remote != ""
	return
}

// RunUpdate downloads pre-built release artifacts and restarts services.
func RunUpdate() (*UpdateResult, error) {
	start := time.Now()
	dir := installDir()
	var sb strings.Builder

	log := func(msg string) {
		sb.WriteString("[" + time.Now().Format("15:04:05") + "] " + msg + "\n")
	}

	prevCommit := strings.TrimSpace(runCmd("git", "-C", dir, "rev-parse", "--short", "HEAD"))
	log("Aktueller Commit: " + prevCommit)

	// Git pull (just for version tracking)
	log("Lade Repository-Status von GitHub...")
	out, err := runCmdOutput("git", "-C", dir, "pull", "--rebase", "origin")
	if err != nil {
		log("WARN git pull: " + out)
	} else {
		log(out)
	}

	newCommit := strings.TrimSpace(runCmd("git", "-C", dir, "rev-parse", "--short", "HEAD"))
	log("Neuer Commit: " + newCommit)

	changedOut := runCmd("git", "-C", dir, "diff", "--name-only", prevCommit, newCommit)
	changedFiles := len(strings.Split(strings.TrimSpace(changedOut), "\n"))
	if strings.TrimSpace(changedOut) == "" {
		changedFiles = 0
	}

	// Download master binary
	log("Lade Master-API Binary herunter...")
	if err := downloadFile(releaseBase+"/master", filepath.Join(dir, "bin/master"), 0755); err != nil {
		return nil, fmt.Errorf("download master: %w", err)
	}
	log("Master-API heruntergeladen.")

	// Download agent binary
	log("Lade Agent Binary herunter...")
	if err := downloadFile(releaseBase+"/agent", filepath.Join(dir, "bin/agent"), 0755); err != nil {
		return nil, fmt.Errorf("download agent: %w", err)
	}
	log("Agent heruntergeladen.")

	// Download and extract frontend
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
		ChangedFiles:   changedFiles,
		Output:         sb.String(),
		Duration:       time.Since(start).Round(time.Second).String(),
		RestartedAt:    time.Now(),
	}, nil
}

func downloadFile(url, dest string, mode os.FileMode) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
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

func installDir() string {
	if d := os.Getenv("INSTALL_DIR"); d != "" {
		return d
	}
	return "/opt/controlpanel"
}

func runCmd(name string, args ...string) string {
	out, _ := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out))
}

func runCmdErr(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func runCmdOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
