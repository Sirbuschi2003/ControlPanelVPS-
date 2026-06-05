package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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
	if out := runCmd("git", "-C", dir, "fetch", "--dry-run", "origin"); out == "" {
		// no-op
	}
	if err2 := runCmdErr("git", "-C", dir, "fetch", "origin"); err2 != nil {
		return false, "", fmt.Errorf("git fetch: %w", err2)
	}
	local := strings.TrimSpace(runCmd("git", "-C", dir, "rev-parse", "HEAD"))
	remote := strings.TrimSpace(runCmd("git", "-C", dir, "rev-parse", "@{u}"))
	latestCommit = remote
	available = local != remote && remote != ""
	return
}

// RunUpdate pulls latest code, rebuilds all components and restarts services.
func RunUpdate() (*UpdateResult, error) {
	start := time.Now()
	dir := installDir()
	var sb strings.Builder

	log := func(msg string) {
		sb.WriteString("[" + time.Now().Format("15:04:05") + "] " + msg + "\n")
	}

	// Current commit before update
	prevCommit := strings.TrimSpace(runCmd("git", "-C", dir, "rev-parse", "--short", "HEAD"))
	log("Aktueller Commit: " + prevCommit)

	// Git pull
	log("Lade aktuelle Version von GitHub...")
	out, err := runCmdOutput("git", "-C", dir, "pull", "--rebase", "origin")
	if err != nil {
		return nil, fmt.Errorf("git pull: %w\n%s", err, out)
	}
	log(out)

	newCommit := strings.TrimSpace(runCmd("git", "-C", dir, "rev-parse", "--short", "HEAD"))
	log("Neuer Commit: " + newCommit)

	// Count changed files
	changedOut := runCmd("git", "-C", dir, "diff", "--name-only", prevCommit, newCommit)
	changedFiles := len(strings.Split(strings.TrimSpace(changedOut), "\n"))
	if strings.TrimSpace(changedOut) == "" {
		changedFiles = 0
	}

	// Rebuild Master API
	log("Baue Master API...")
	out, err = runCmdOutput("go", "build", "-ldflags=-w -s", "-o", filepath.Join(dir, "bin/master"), "./cmd/server")
	if err != nil {
		// Try with absolute go path
		goPath := findGo()
		out, err = runCmdInDir(filepath.Join(dir, "master"), goPath, "build", "-ldflags=-w -s", "-o", filepath.Join(dir, "bin/master"), "./cmd/server")
		if err != nil {
			return nil, fmt.Errorf("build master: %w\n%s", err, out)
		}
	}
	log("Master API erfolgreich gebaut.")

	// Rebuild Agent
	log("Baue Agent...")
	goPath := findGo()
	out, err = runCmdInDir(filepath.Join(dir, "agent"), goPath, "build", "-ldflags=-w -s", "-o", filepath.Join(dir, "bin/agent"), "./cmd/agent")
	if err != nil {
		return nil, fmt.Errorf("build agent: %w\n%s", err, out)
	}
	log("Agent erfolgreich gebaut.")

	// Build frontend
	log("Baue Frontend...")
	out, err = runCmdInDir(filepath.Join(dir, "frontend"), "npm", "ci", "--silent")
	if err != nil {
		log("WARN: npm ci: " + out)
	}
	out, err = runCmdInDir(filepath.Join(dir, "frontend"), "npm", "run", "build")
	if err != nil {
		return nil, fmt.Errorf("build frontend: %w\n%s", err, out)
	}
	log("Frontend erfolgreich gebaut.")

	// Restart services
	log("Starte Dienste neu...")
	for _, svc := range []string{"cpanel-master", "cpanel-agent"} {
		if out, err := runCmdOutput("systemctl", "restart", svc); err != nil {
			log("WARN: restart " + svc + ": " + out)
		} else {
			log(svc + " neu gestartet.")
		}
	}

	result := &UpdateResult{
		PreviousCommit: prevCommit,
		NewCommit:      newCommit,
		ChangedFiles:   changedFiles,
		Output:         sb.String(),
		Duration:       time.Since(start).Round(time.Second).String(),
		RestartedAt:    time.Now(),
	}
	return result, nil
}

func installDir() string {
	if d := os.Getenv("INSTALL_DIR"); d != "" {
		return d
	}
	return "/opt/controlpanel"
}

func findGo() string {
	for _, p := range []string{"/usr/local/go/bin/go", "/usr/bin/go"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "go"
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

func runCmdInDir(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
