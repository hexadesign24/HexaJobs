package client

import (
	"reflect"
	"testing"
)

func TestBrowserArgumentsAcrossPlatforms(t *testing.T) {
	url := "https://example.com/jobs?a=1&b=two%20words"
	for _, tc := range []struct {
		os, command string
		args        []string
	}{
		{"linux", "xdg-open", []string{url}}, {"darwin", "open", []string{url}}, {"windows", "rundll32", []string{"url.dll,FileProtocolHandler", url}},
	} {
		name, args, err := browserCommand(tc.os, url)
		if err != nil || name != tc.command || !reflect.DeepEqual(args, tc.args) {
			t.Fatalf("%s: %s %v %v", tc.os, name, args, err)
		}
	}
	if _, _, err := browserCommand("unknown", url); err == nil {
		t.Fatal("unsupported OS accepted")
	}
}

func TestBrowserRejectsUnsafeLinks(t *testing.T) {
	for _, url := range []string{"javascript:alert(1)", "file:///etc/passwd", "--flag", "https://user:pass@example.com", "https://example.com\ncommand", "https://", " https://example.com", "https://ex\u202eample.com", "https://example.com/a b"} {
		if err := ValidateURL(url); err == nil {
			t.Errorf("accepted unsafe URL %q", url)
		}
	}
}
