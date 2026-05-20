// Package upgrade replaces the running claude-task binary in place with the
// latest release published to GitHub. It mirrors install.sh: same asset naming,
// same release source, but applied to the currently running executable.
package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repo   = "liunuozhi/claude-task"
	binary = "claude-task"
)

var httpClient = &http.Client{Timeout: 2 * time.Minute}

// Run upgrades the running binary to the latest GitHub release. current is the
// version compiled into this build; a "dev" build always re-downloads.
func Run(current string) error {
	tag, err := latestTag()
	if err != nil {
		return fmt.Errorf("checking latest release: %w", err)
	}

	fmt.Printf("Current: %s\n", current)
	fmt.Printf("Latest:  %s\n", tag)

	if current != "dev" && strings.TrimPrefix(current, "v") == strings.TrimPrefix(tag, "v") {
		fmt.Println("Already up to date.")
		return nil
	}

	asset := fmt.Sprintf("%s_%s_%s.tar.gz", binary, runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, asset)

	fmt.Printf("Downloading %s...\n", asset)
	bin, err := downloadBinary(url)
	if err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current executable: %w", err)
	}
	// Replace the real file, not a symlink pointing at it.
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	fmt.Printf("Replacing %s\n", exe)
	if err := replaceExecutable(exe, bin); err != nil {
		return err
	}

	fmt.Printf("Upgraded to %s.\n", tag)
	return nil
}

// latestTag returns the tag_name of the newest GitHub release.
func latestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", binary)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %s", resp.Status)
	}

	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no release found")
	}
	return rel.TagName, nil
}

// downloadBinary fetches the release tarball and returns the extracted binary.
func downloadBinary(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", binary)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s (no release asset for %s/%s?)", resp.Status, runtime.GOOS, runtime.GOARCH)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == binary {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", binary)
}

// replaceExecutable atomically swaps the file at path with data: write a temp
// file in the same directory (so rename stays on one filesystem), then rename
// over the target. The running process keeps its open inode, so this is safe.
func replaceExecutable(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".claude-task-upgrade-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w (try: sudo claude-task upgrade)", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("cannot replace %s: %w (try: sudo claude-task upgrade)", path, err)
	}
	return nil
}
