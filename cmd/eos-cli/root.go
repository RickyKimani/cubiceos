package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rickykimani/cubiceos/internal/tui"
	"github.com/rickykimani/cubiceos/internal/web"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var httpMode bool
	var update bool

	cmd := &cobra.Command{
		Use:   "eos-cli",
		Short: "Interactive EOS solver",
		Long: `CubicEOS is a solver (in molar volume) for cubic equations of state.
By default, running 'eos-cli' launches the interactive terminal UI.

Use '--http' to start the web UI instead.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if update {
				return doUpdate(cmd.Version)
			}
			// Kick off background update check before launching anything.
			go checkForUpdateAsync(cmd.Version)
			if httpMode {
				web.Run()
				return nil
			}
			return tui.Run()
		},
	}

	cmd.Flags().BoolVar(&httpMode, "http", false, "Launch the web UI instead of the TUI")
	cmd.Flags().BoolVarP(&update, "update", "u", false, "Update eos-cli to the latest version")

	return cmd
}

func fetchLatestTag(ctx context.Context) (string, error) {
	url := "https://api.github.com/repos/rickykimani/cubiceos/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	var tags []struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", err
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found")
	}
	return tags[0].Name, nil
}

func doUpdate(current string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	latest, err := fetchLatestTag(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch latest version: %w", err)
	}

	cur := extractVersionToken(current)
	if cur == "" {
		return fmt.Errorf("could not parse current version: %q", current)
	}
	if cur == latest {
		fmt.Println("Already up to date")
		return nil
	}

	fmt.Printf("Updating %s → %s...\n", cur, latest)

	cmd := exec.Command("go", "install",
		fmt.Sprintf("github.com/rickykimani/cubiceos/cmd/eos-cli@%s", latest),
	)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Println("Update complete")
	return nil
}

func checkForUpdateAsync(currentRaw string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	latest, err := fetchLatestTag(ctx)
	if err != nil || latest == "" {
		return
	}

	cur := extractVersionToken(currentRaw)
	if cur == "" || cur == latest {
		return
	}

	fmt.Fprintf(os.Stderr,
		"A new version is available: %s (you have %s). Run 'eos-cli -u' to update.\n",
		latest, cur,
	)
}

func extractVersionToken(s string) string {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if strings.HasPrefix(last, "v") {
		return last
	}
	for _, p := range parts {
		if strings.HasPrefix(p, "v") {
			return p
		}
	}
	return ""
}
