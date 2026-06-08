package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/version"
)

const githubAPIBase = "https://api.github.com"

type PanelInfo struct {
	Commit     string `json:"commit"`
	Date       string `json:"date"`
	InstallDir string `json:"install_dir"`
}

type PanelUpdateCheck struct {
	Available     bool   `json:"available"`
	CurrentCommit string `json:"current_commit"`
	LatestCommit  string `json:"latest_commit"`
	PublishedAt   string `json:"published_at"`
}

type PanelUpdateResult struct {
	PreviousCommit string `json:"previous_commit"`
	NewCommit      string `json:"new_commit"`
	Duration       string `json:"duration"`
	RestartedAt    string `json:"restarted_at"`
}

type githubRelease struct {
	TagName     string         `json:"tag_name"`
	PublishedAt time.Time      `json:"published_at"`
	Body        string         `json:"body"`
	Assets      []githubAsset  `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// PanelUpdateService handles self-updates of the panel software.
type PanelUpdateService struct {
	installDir string
	githubRepo string
}

func NewPanelUpdateService(installDir, githubRepo string) *PanelUpdateService {
	return &PanelUpdateService{
		installDir: installDir,
		githubRepo: githubRepo,
	}
}

func (s *PanelUpdateService) GetInfo() *PanelInfo {
	return &PanelInfo{
		Commit:     version.Commit,
		Date:       version.Date,
		InstallDir: s.installDir,
	}
}

func (s *PanelUpdateService) CheckUpdate(ctx context.Context) (*PanelUpdateCheck, error) {
	release, err := s.fetchLatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	latestCommit := s.extractCommitFromBody(release.Body)

	available := false
	if version.Commit == "dev" {
		available = false
	} else if latestCommit != "" && latestCommit != version.Commit {
		available = true
	} else if version.Date != "unknown" {
		buildTime, err := time.Parse(time.RFC3339, version.Date)
		if err == nil {
			available = release.PublishedAt.After(buildTime)
		}
	}

	return &PanelUpdateCheck{
		Available:     available,
		CurrentCommit: shortCommit(version.Commit),
		LatestCommit:  shortCommit(latestCommit),
		PublishedAt:   release.PublishedAt.Format(time.RFC3339),
	}, nil
}

func (s *PanelUpdateService) RunUpdate(ctx context.Context) (*PanelUpdateResult, error) {
	start := time.Now()

	release, err := s.fetchLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}

	masterURL, frontendURL := s.findAssetURLs(release)
	if masterURL == "" {
		return nil, fmt.Errorf("master binary not found in release assets")
	}

	previousCommit := version.Commit

	if err := s.downloadBinary(ctx, masterURL); err != nil {
		return nil, fmt.Errorf("download master binary: %w", err)
	}

	if frontendURL != "" {
		if err := s.downloadAndExtractFrontend(ctx, frontendURL); err != nil {
			return nil, fmt.Errorf("download frontend: %w", err)
		}
	}

	latestCommit := s.extractCommitFromBody(release.Body)
	result := &PanelUpdateResult{
		PreviousCommit: shortCommit(previousCommit),
		NewCommit:      shortCommit(latestCommit),
		Duration:       time.Since(start).Round(time.Second).String(),
		RestartedAt:    time.Now().Format(time.RFC3339),
	}

	// Restart after response is sent — goroutine exits when process is killed by systemd
	go func() {
		time.Sleep(2 * time.Second)
		_ = exec.Command("systemctl", "restart", "cpanel-master", "cpanel-frontend").Run()
	}()

	return result, nil
}

func (s *PanelUpdateService) fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/tags/latest", githubAPIBase, s.githubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ControlPanelVPS")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &release, nil
}

func (s *PanelUpdateService) extractCommitFromBody(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "commit:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "commit:"))
		}
	}
	return ""
}

func (s *PanelUpdateService) findAssetURLs(release *githubRelease) (masterURL, frontendURL string) {
	for _, a := range release.Assets {
		switch a.Name {
		case "master":
			masterURL = a.BrowserDownloadURL
		case "frontend.tar.gz":
			frontendURL = a.BrowserDownloadURL
		}
	}
	return
}

func (s *PanelUpdateService) downloadBinary(ctx context.Context, url string) error {
	tmpPath := filepath.Join(os.TempDir(), "cpanel-master.new")
	if err := downloadFile(ctx, url, tmpPath); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}
	dest := filepath.Join(s.installDir, "bin", "master")
	// Atomic rename: replaces inode, safe for running binaries
	return os.Rename(tmpPath, dest)
}

func (s *PanelUpdateService) downloadAndExtractFrontend(ctx context.Context, url string) error {
	tmpPath := filepath.Join(os.TempDir(), "cpanel-frontend.tar.gz")
	if err := downloadFile(ctx, url, tmpPath); err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	destDir := filepath.Join(s.installDir, "frontend-standalone")
	if err := extractTarGz(tmpPath, destDir); err != nil {
		return err
	}

	// Fix ownership so cpanel user can read the new files
	_ = exec.Command("chown", "-R", "cpanel:cpanel", destDir).Run()
	return nil
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ControlPanelVPS")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func extractTarGz(src, destDir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Sanitize path to prevent directory traversal
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") {
			continue
		}

		target := filepath.Join(destDir, cleanName)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func shortCommit(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}
