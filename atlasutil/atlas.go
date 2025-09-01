package atlasutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const atlasVersion = "0.36.0"

func download(ctx context.Context, atlasPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://release.ariga.io/atlas/atlas-%s-%s-v%s", runtime.GOOS, runtime.GOARCH, atlasVersion), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download atlas: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download atlas: %s", resp.Status)
	}

	out, err := os.Create(atlasPath)
	if err != nil {
		return fmt.Errorf("failed to create atlas binary: %w", err)
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to copy atlas binary: %w", err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close atlas binary: %w", err)
	}

	if err := os.Chmod(atlasPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod atlas binary: %w", err)
	}

	return nil
}

func Migrate(ctx context.Context, uri string, baseline string) error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %w", err)
	}

	atlasDir := filepath.Join(cacheDir, "atlas")
	if err := os.MkdirAll(atlasDir, 0755); err != nil {
		return fmt.Errorf("failed to create atlas cache directory: %w", err)
	}

	atlasPath := filepath.Join(atlasDir, fmt.Sprintf("atlas-%s-%s-v%s", runtime.GOOS, runtime.GOARCH, atlasVersion))
	if _, err := os.Stat(atlasPath); os.IsNotExist(err) {
		if err := download(ctx, atlasPath); err != nil {
			return fmt.Errorf("failed to download atlas: %w", err)
		}
	}

	koenv, ok := os.LookupEnv("KO_DATA_PATH")
	if !ok {
		koenv = "/kodata"
	}

	migrationDir := filepath.Join(koenv, "migrations")

	baseCommand := []string{
		atlasPath,
		"migrate",
		"apply",
		"--url", uri,
		"--dir", fmt.Sprintf("file://%s", migrationDir),
	}

	// First try without baseline
	//nolint:gosec // false positive for G204
	cmd := exec.CommandContext(ctx, baseCommand[0], baseCommand[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := string(exitErr.Stderr)
			// Only retry with baseline if a database is not clean error
			if !strings.Contains(stderr, "connected database is not clean") {
				return fmt.Errorf("failed to run atlas migrate: %w", err)
			}
		}

		if baseline == "" {
			return fmt.Errorf("failed to run atlas migrate: %w", err)
		}

		//nolint:gosec // false positive for G204
		cmd = exec.CommandContext(ctx, baseCommand[0], append(baseCommand[1:], "--baseline", baseline)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run atlas migrate with baseline: %w", err)
		}
	}

	return nil
}
