package views

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"hexajobs.dev/hexajobs-cli/internal/models"
)

func testStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().Bold(true),
		Muted: lipgloss.NewStyle(),
		Text:  lipgloss.NewStyle(),
		Border: lipgloss.NewStyle().Border(lipgloss.NormalBorder()),
	}
}

func TestLogoShownOnDashboardWhenRoom(t *testing.T) {
	s := testStyles()
	out := Dashboard("> go", models.MarketReport{}, 56, 18, "en", s)
	if !strings.Contains(out, "HEX") {
		t.Fatal("expected HEX ascii art on 56x18 dashboard")
	}
	if !strings.Contains(out, LogoTagline) {
		t.Fatal("expected tagline on dashboard")
	}
}

func TestLogoHiddenOnSmallDashboard(t *testing.T) {
	s := testStyles()
	if ShowLogo(30, 10) {
		t.Fatal("ShowLogo should be false for 30x10")
	}
	out := Dashboard("> go", models.MarketReport{}, 30, 10, "en", s)
	if strings.Contains(out, LogoTagline) {
		t.Fatal("tagline should be hidden on small dashboard")
	}
}

func TestLogoLinesWithinWidth(t *testing.T) {
	s := testStyles()
	for _, line := range Logo(54, s) {
		if w := ansi.StringWidth(ansi.Strip(line)); w > 54 {
			t.Fatalf("logo line width %d exceeds 54: %q", w, line)
		}
	}
}
