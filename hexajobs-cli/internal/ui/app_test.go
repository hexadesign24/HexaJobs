package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"hexajobs.dev/hexajobs-cli/internal/client"
	"hexajobs.dev/hexajobs-cli/internal/models"
)

type fakeEngine struct {
	fetch  func(context.Context, string, string, string) ([]models.JobListing, error)
	purges int
}

func (e *fakeEngine) FetchJobs(c context.Context, k, r, cat string) ([]models.JobListing, error) {
	if e.fetch != nil {
		return e.fetch(c, k, r, cat)
	}
	return nil, nil
}
func (e *fakeEngine) GetMarketPulse(string) models.MarketReport {
	return models.MarketReport{AverageRate: "USD 50.00/hour", Sentiment: "STABLE"}
}
func (e *fakeEngine) PurgeExpiredCache() int { e.purges++; return 2 }

func testModel(t *testing.T, e models.EngineContract) Model {
	t.Helper()
	m := NewModel(e, WithServices(Services{
		LoadPreferences: func() (Preferences, error) { return DefaultPreferences(), nil }, SavePreferences: func(Preferences) error { return nil }, History: func() ([]client.HistoryEntry, error) { return nil, nil }, Log: func(models.JobListing, string) error { return nil }, OpenBrowser: func(context.Context, string) error { return nil },
	}))
	t.Cleanup(m.Close)
	m.ready = true
	m.Loading = false
	return m
}

func key(m Model, value string) (Model, tea.Cmd) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
	switch value {
	case "enter":
		msg.Type = tea.KeyEnter
	case "esc":
		msg.Type = tea.KeyEsc
	case "tab":
		msg.Type = tea.KeyTab
	case "up":
		msg.Type = tea.KeyUp
	case "down":
		msg.Type = tea.KeyDown
	case "right":
		msg.Type = tea.KeyRight
	case "ctrl+c":
		msg.Type = tea.KeyCtrlC
	}
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

// Application work is the last Cmd in these batches; animation timers are
// intentionally left to Bubble Tea rather than executed by synchronous tests.
func work(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		return work(batch[len(batch)-1])
	}
	return msg
}
func deliver(m Model, msg tea.Msg) (Model, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}
func fixture() models.JobListing {
	return models.JobListing{ID: "1", Title: "Go Engineer", Company: "Acme", Platform: "test", ApplyURL: "https://example.com/job", Region: "Worldwide", RiskStatus: "GREEN", GhostingRate: -1, Category: "IT", CompensationUSD: 60, CompensationPeriod: "hour", WinRate: 80, SkillsMatch: 0.8}
}

func TestBootAndSearchRunOnlyInsideCommands(t *testing.T) {
	calls := 0
	e := &fakeEngine{fetch: func(_ context.Context, k, r, c string) ([]models.JobListing, error) {
		calls++
		if k != "go" || r != "" {
			t.Errorf("query %q %q", k, r)
		}
		a := fixture()
		b := a
		b.ID = "hidden"
		b.GhostingRate = 0.9
		return []models.JobListing{a, b}, errors.New("secondary offline")
	}}
	m := testModel(t, e)
	if e.purges != 0 {
		t.Fatal("constructor performed engine I/O")
	}
	cmd := m.Init()
	if e.purges != 0 {
		t.Fatal("Init performed I/O before Cmd ran")
	}
	m, _ = deliver(m, work(cmd))
	if e.purges != 1 || !m.ready {
		t.Fatal("boot failed")
	}
	m.input.SetValue("go")
	m, cmd = key(m, "enter")
	if calls != 0 || !m.Loading || m.State != ViewResults {
		t.Fatalf("search not scheduled correctly %+v", m)
	}
	m, _ = deliver(m, work(cmd))
	if calls != 1 || m.Loading || len(m.Results) != 1 || !strings.Contains(m.StatusMessage, "Partial") {
		t.Fatalf("partial search %+v", m.Results)
	}
}

func TestTypingFocusNavigationAndStaleResponses(t *testing.T) {
	m := testModel(t, &fakeEngine{})
	m, _ = key(m, "q")
	m, _ = key(m, "j")
	if m.Query != "qj" {
		t.Fatal("input shortcuts stole letters", m.Query)
	}
	m, _ = key(m, "tab")
	m, _ = key(m, "down")
	m, cmd := key(m, "enter")
	if m.State != ViewRadar || !m.Loading || cmd == nil {
		t.Fatal("sidebar radar failed")
	}
	oldID := m.requestID
	m, _ = key(m, "esc")
	m, _ = deliver(m, fetchMsg{id: oldID, jobs: []models.JobListing{fixture()}, radar: true})
	if m.State != ViewDashboard || m.Loading || len(m.radarJobs) != 0 {
		t.Fatal("late request replaced navigation state")
	}
	m, _ = key(m, "tab")
	_, cmd = key(m, "q")
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("main menu q did not quit")
	}
}

func TestSearchCancellationAndWeb3BountyContract(t *testing.T) {
	var categories []string
	var captured context.Context
	e := &fakeEngine{fetch: func(ctx context.Context, k, r, cat string) ([]models.JobListing, error) {
		captured = ctx
		categories = append(categories, cat)
		j := fixture()
		j.ID = cat
		j.Category = cat
		return []models.JobListing{j}, nil
	}}
	m := testModel(t, e)
	m.mainFocus = false
	m.sidebarCursor = 2
	m, cmd := key(m, "enter")
	m, _ = deliver(m, work(cmd))
	if strings.Join(categories, ",") != "Web3,Bounty" || len(m.Results) != 2 {
		t.Fatal("opportunity routing", categories)
	}
	if !errors.Is(captured.Err(), context.Canceled) {
		t.Fatal("fetch context not released")
	}
	m, cmd = key(m, "r")
	id := m.requestID
	m, _ = key(m, "esc")
	if m.requestID == id || m.Loading {
		t.Fatal("cancel did not invalidate request")
	}
	_ = cmd
}

func TestScamShieldBlocksApplyAndAsyncActionsPreserveHistory(t *testing.T) {
	opened, logged := 0, 0
	m := testModel(t, &fakeEngine{})
	m.State = ViewResults
	m.Results = []models.JobListing{fixture()}
	m.services.OpenBrowser = func(context.Context, string) error { opened++; return nil }
	m.services.Log = func(_ models.JobListing, action string) error {
		logged++
		if action != "opened" && action != "saved" && action != "pitch" {
			t.Fatal(action)
		}
		return nil
	}
	m.Results[0].Description = "bank account rental"
	m, cmd := key(m, "a")
	if cmd != nil || opened != 0 || !strings.Contains(m.StatusMessage, "blocked") {
		t.Fatal("red listing not blocked")
	}
	m.Preferences.ScamShield = false
	m, cmd = key(m, "a")
	if opened != 0 || !m.actionBusy {
		t.Fatal("browser ran in Update")
	}
	m, _ = deliver(m, work(cmd))
	if opened != 1 || logged != 1 || m.actionBusy {
		t.Fatal("browser/log action failed")
	}
	m, cmd = key(m, "p")
	m, _ = deliver(m, work(cmd))
	if !strings.Contains(m.overlayText, "Subject:") || logged != 2 {
		t.Fatal("pitch overlay failed")
	}
	m, _ = key(m, "esc")
	if m.overlayText != "" || m.State != ViewResults {
		t.Fatal("overlay Esc lost results")
	}
	m.Results[0].IsMock = true
	m, cmd = key(m, "a")
	if cmd != nil {
		t.Fatal("demo opened browser")
	}
}

func TestFailedBrowserDoesNotLogSuccessAndLatePitchDoesNotNavigate(t *testing.T) {
	m := testModel(t, &fakeEngine{})
	m.State = ViewResults
	m.Results = []models.JobListing{fixture()}
	logs := 0
	m.services.OpenBrowser = func(context.Context, string) error { return errors.New("no browser") }
	m.services.Log = func(models.JobListing, string) error { logs++; return nil }
	m, cmd := key(m, "a")
	m, _ = deliver(m, work(cmd))
	if logs != 0 || !strings.Contains(m.StatusMessage, "no browser") {
		t.Fatal("failed browser logged as success")
	}
	m, cmd = key(m, "p")
	m, _ = key(m, "esc")
	m, _ = deliver(m, work(cmd))
	if m.State != ViewDashboard || m.overlayText != "" {
		t.Fatal("late pitch stole screen")
	}
}

func TestPreferencesWritesCoalesceAndHistoryIsAsync(t *testing.T) {
	m := testModel(t, &fakeEngine{})
	m.State = ViewSettings
	saves := []Preferences{}
	m.services.SavePreferences = func(p Preferences) error { saves = append(saves, p); return nil }
	m, first := key(m, "right")
	if m.Preferences.Language != "id" || len(saves) != 0 {
		t.Fatal("preference not changed asynchronously")
	}
	m, second := key(m, "right")
	if second != nil || m.Preferences.Language != "jp" {
		t.Fatal("inflight save not coalesced")
	}
	m, next := deliver(m, work(first))
	m, _ = deliver(m, work(next))
	if len(saves) != 2 || saves[1].Language != "jp" {
		t.Fatal("latest preferences not persisted")
	}
	m.mainFocus = false
	m.sidebarCursor = 3
	reads := 0
	m.services.History = func() ([]client.HistoryEntry, error) {
		reads++
		return []client.HistoryEntry{{Job: fixture(), Action: "saved"}}, nil
	}
	m, cmd := key(m, "enter")
	if reads != 0 || m.State != ViewHistory || !m.Loading {
		t.Fatal("history ran in Update")
	}
	m, _ = deliver(m, work(cmd))
	if reads != 1 || len(m.history) != 1 {
		t.Fatal("history not loaded")
	}
}

func TestAllViewsStayWithinTerminalBounds(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {100, 30}, {120, 40}, {60, 18}, {30, 8}} {
		for _, lang := range []string{"en", "id", "jp"} {
			for state := ViewDashboard; state <= ViewHistory; state++ {
				t.Run(fmt.Sprintf("%dx%d/%s/%d", size[0], size[1], lang, state), func(t *testing.T) {
					m := testModel(t, &fakeEngine{})
					m.State = state
					m.Preferences.Language = lang
					m.Results = []models.JobListing{fixture(), fixture(), fixture()}
					m.Results[0].Title = strings.Repeat("設計者🚀", 80) + "\x1b]52;c;bad\a"
					m.pulse = models.MarketReport{AverageRate: "USD 50.00/hour", Sentiment: "STABLE"}
					m.radarJobs = m.Results
					m.history = []client.HistoryEntry{{Job: fixture(), Action: "saved"}}
					m, _ = deliver(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
					render := m.View()
					lines := strings.Split(render, "\n")
					if len(lines) > size[1] {
						t.Fatalf("height %d > %d", len(lines), size[1])
					}
					for i, line := range lines {
						if width := ansi.StringWidth(line); width > size[0] {
							t.Fatalf("line %d width %d > %d: %q", i, width, size[0], line)
						}
					}
					if strings.Contains(render, "]52;") {
						t.Fatal("external OSC sequence rendered")
					}
					if size[0] >= 80 && !strings.Contains(ansi.Strip(render), "HEXAJOBS.DEV") {
						t.Fatal("missing toolbar")
					}
				})
			}
		}
	}
}

func TestOverlayWrapsOnResize(t *testing.T) {
	m := testModel(t, &fakeEngine{})
	m.openOverlay("Pitch", strings.Repeat("Long draft text 日本語 ", 200))
	m, _ = deliver(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.viewport.TotalLineCount() <= m.viewport.Height {
		t.Fatal("long overlay not scrollable")
	}
	for _, line := range strings.Split(m.View(), "\n") {
		if ansi.StringWidth(line) > 80 {
			t.Fatal("overlay overflow")
		}
	}
}
