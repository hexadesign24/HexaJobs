package client

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"
)

// ValidateURL rejects shell-like schemes, embedded credentials and controls.
// The URL is always a single argument, never interpolated into a shell command.
func ValidateURL(raw string) error {
	if len(raw) > 8192 || strings.TrimSpace(raw) != raw {
		return errors.New("invalid link")
	}
	for _, r := range raw {
		if unicode.IsControl(r) || unicode.IsSpace(r) || unicode.Is(unicode.Cf, r) {
			return errors.New("invalid link characters")
		}
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return errors.New("only absolute HTTP(S) links without credentials are allowed")
	}
	return nil
}

func browserCommand(goos, raw string) (string, []string, error) {
	if err := ValidateURL(raw); err != nil {
		return "", nil, err
	}
	switch goos {
	case "linux":
		return "xdg-open", []string{raw}, nil
	case "darwin":
		return "open", []string{raw}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", raw}, nil
	default:
		return "", nil, fmt.Errorf("browser launch unsupported on %s", goos)
	}
}

func OpenBrowser(raw string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return OpenBrowserContext(ctx, raw)
}

func OpenBrowserContext(ctx context.Context, raw string) error {
	name, args, err := browserCommand(runtime.GOOS, raw)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	configureBrowserCommand(cmd)
	// stdout/stderr remain nil: helpers cannot corrupt the alternate-screen TUI.
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("browser launcher %s failed: %w", name, err)
	}
	return nil
}
