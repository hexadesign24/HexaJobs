package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"hexajobs.dev/hexajobs-cli/internal/client"
	"hexajobs.dev/hexajobs-cli/internal/models"
	"hexajobs.dev/hexajobs-cli/internal/security"
	"hexajobs.dev/hexajobs-cli/internal/ui/views"
)

type ViewState int

const (
	ViewDashboard ViewState = iota
	ViewResults
	ViewRadar
	ViewSettings
	ViewDonate
	ViewHistory
)

type Services struct {
	Bootstrap       func() (models.EngineContract, error)
	OpenBrowser     func(context.Context, string) error
	Log             func(models.JobListing, string) error
	History         func() ([]client.HistoryEntry, error)
	LoadPreferences func() (Preferences, error)
	SavePreferences func(Preferences) error
}

type Option func(*Model)

func WithServices(services Services) Option {
	return func(m *Model) {
		if services.Bootstrap != nil {
			m.services.Bootstrap = services.Bootstrap
		}
		if services.OpenBrowser != nil {
			m.services.OpenBrowser = services.OpenBrowser
		}
		if services.Log != nil {
			m.services.Log = services.Log
		}
		if services.History != nil {
			m.services.History = services.History
		}
		if services.LoadPreferences != nil {
			m.services.LoadPreferences = services.LoadPreferences
		}
		if services.SavePreferences != nil {
			m.services.SavePreferences = services.SavePreferences
		}
	}
}
func WithSponsorURL(url string) Option { return func(m *Model) { m.sponsorURL = url } }
func WithDemo() Option                 { return func(m *Model) { m.demo = true } }

type Model struct {
	Engine        models.EngineContract
	Query         string
	Results       []models.JobListing
	Cursor        int
	Loading       bool
	StatusMessage string
	State         ViewState
	Preferences   Preferences
	FatalErr      error

	width, height                              int
	mainFocus                                  bool
	sidebarCursor, settingsCursor              int
	input                                      textinput.Model
	spinner                                    spinner.Model
	viewport                                   viewport.Model
	styles                                     views.Styles
	pulse                                      models.MarketReport
	history                                    []client.HistoryEntry
	radarJobs                                  []models.JobListing
	overlayTitle, overlayText                  string
	category                                   string
	ctx                                        context.Context
	cancel                                     context.CancelFunc
	requestCancel                              context.CancelFunc
	requestID, navigationID                    uint64
	services                                   Services
	ready, actionBusy, prefsSaving, prefsDirty bool
	sponsorURL                                 string
	demo                                       bool
}

type bootMsg struct {
	engine        models.EngineContract
	prefs         Preferences
	pulse         models.MarketReport
	purged        int
	err, prefsErr error
}
type fetchMsg struct {
	id    uint64
	jobs  []models.JobListing
	pulse models.MarketReport
	err   error
	radar bool
}
type historyMsg struct {
	id      uint64
	entries []client.HistoryEntry
	err     error
}
type actionMsg struct {
	navigation    uint64
	action, pitch string
	err           error
}
type preferencesMsg struct{ err error }

// NewModel is pure: disk, engine calls and process launches only happen in Cmds.
func NewModel(engine models.EngineContract, options ...Option) Model {
	ctx, cancel := context.WithCancel(context.Background())
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "Go, Solidity, design..."
	input.CharLimit = 180
	input.Width = 44
	input.Focus()
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	m := Model{Engine: engine, Preferences: DefaultPreferences(), width: 80, height: 24, mainFocus: true, input: input, spinner: spin, viewport: viewport.New(50, 14), styles: NewStyles(), ctx: ctx, cancel: cancel, Loading: true, StatusMessage: "Starting engine..."}
	m.services = Services{
		OpenBrowser: client.OpenBrowserContext,
		Log: func(j models.JobListing, a string) error {
			t, err := client.DefaultTracker()
			if err != nil {
				return err
			}
			return t.Log(j, a)
		},
		History: func() ([]client.HistoryEntry, error) {
			t, err := client.DefaultTracker()
			if err != nil {
				return nil, err
			}
			return t.Read()
		},
		LoadPreferences: LoadPreferences, SavePreferences: SavePreferences,
	}
	for _, option := range options {
		option(&m)
	}
	m.resize()
	return m
}

func (m Model) Close() { m.cancel() }

func (m Model) Init() tea.Cmd {
	engine, services := m.Engine, m.services
	return tea.Batch(m.spinner.Tick, textinput.Blink, func() tea.Msg {
		msg := bootMsg{engine: engine, prefs: DefaultPreferences()}
		if msg.engine == nil {
			if services.Bootstrap == nil {
				msg.err = errors.New("engine unavailable")
			} else {
				msg.engine, msg.err = services.Bootstrap()
			}
		}
		if msg.err == nil && msg.engine != nil {
			msg.purged = msg.engine.PurgeExpiredCache()
			msg.pulse = msg.engine.GetMarketPulse("")
		}
		msg.prefs, msg.prefsErr = services.LoadPreferences()
		return msg
	})
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width = max(1, msg.Width)
		m.height = max(1, msg.Height)
		m.resize()
		return m, nil
	case bootMsg:
		m.Engine = msg.engine
		m.ready = msg.err == nil && msg.engine != nil
		m.Loading = false
		m.FatalErr = msg.err
		m.Preferences = msg.prefs
		m.pulse = msg.pulse
		m.StatusMessage = fmt.Sprintf("Ready · %d expired cache files removed", msg.purged)
		if msg.err != nil {
			m.StatusMessage = "Engine unavailable: " + msg.err.Error()
		}
		if msg.prefsErr != nil {
			m.StatusMessage = "Preferences: " + msg.prefsErr.Error() + "; defaults active"
		}
		return m, nil
	case fetchMsg:
		if msg.id != m.requestID {
			return m, nil
		}
		m.Loading = false
		m.requestCancel = nil
		m.pulse = msg.pulse
		filtered := security.FilterGhosting(msg.jobs)
		hidden := len(msg.jobs) - len(filtered)
		if msg.radar {
			m.radarJobs = filtered
		} else {
			m.Results = filterRegion(filtered, m.Preferences.Region)
			m.Cursor = 0
		}
		count := len(m.Results)
		if msg.radar {
			count = len(m.radarJobs)
		}
		m.StatusMessage = fmt.Sprintf("%d listings · %d hidden by anti-ghosting", count, hidden)
		if msg.err != nil {
			m.StatusMessage += " · Partial/unavailable: " + msg.err.Error()
		}
		return m, nil
	case historyMsg:
		if msg.id != m.requestID {
			return m, nil
		}
		m.Loading = false
		m.history = msg.entries
		m.Cursor = 0
		if msg.err != nil {
			m.StatusMessage = "History: " + msg.err.Error()
		} else {
			m.StatusMessage = fmt.Sprintf("%d local events · opened does not mean submitted", len(m.history))
		}
		return m, nil
	case actionMsg:
		m.actionBusy = false
		if msg.err != nil {
			m.StatusMessage = msg.err.Error()
		} else {
			switch msg.action {
			case "opened":
				m.StatusMessage = "Browser opened · no application submitted"
			case "saved":
				m.StatusMessage = "Listing saved to local history"
			case "pitch":
				m.StatusMessage = "Local draft ready · edit evidence before sending"
			case "sponsor":
				m.StatusMessage = "Sponsor page opened"
			}
		}
		if msg.pitch != "" && msg.navigation == m.navigationID {
			m.openOverlay(views.Tr(m.Preferences.Language, "AI Pitch / local draft"), msg.pitch)
		}
		return m, nil
	case preferencesMsg:
		m.prefsSaving = false
		if msg.err != nil {
			m.StatusMessage = "Preferences apply this session; save failed: " + msg.err.Error()
		} else {
			m.StatusMessage = views.Tr(m.Preferences.Language, "Settings saved")
		}
		if m.prefsDirty {
			m.prefsDirty = false
			cmd := m.savePreferences()
			return m, cmd
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.Loading || m.actionBusy {
			return m, cmd
		}
		return m, nil
	case tea.KeyMsg:
		return m.key(msg)
	}
	if m.State == ViewDashboard && m.mainFocus && m.overlayText == "" {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(message)
		return m, cmd
	}
	return m, nil
}

func (m Model) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" || (key == "q" && !m.mainFocus && m.State == ViewDashboard) {
		m.Close()
		return m, tea.Quit
	}
	if key == "esc" {
		if m.overlayText != "" {
			m.overlayText = ""
			m.overlayTitle = ""
			return m, nil
		}
		m.stopRequest()
		m.navigationID++
		m.State = ViewDashboard
		m.sidebarCursor = 0
		m.mainFocus = true
		m.category = ""
		cmd := m.input.Focus()
		return m, cmd
	}
	if key == "tab" {
		m.mainFocus = !m.mainFocus
		if m.mainFocus && m.State == ViewDashboard {
			cmd := m.input.Focus()
			return m, cmd
		}
		m.input.Blur()
		return m, nil
	}
	if !m.mainFocus {
		switch key {
		case "up", "k":
			m.sidebarCursor = (m.sidebarCursor + 5) % 6
		case "down", "j":
			m.sidebarCursor = (m.sidebarCursor + 1) % 6
		case "enter":
			return m.chooseMenu()
		}
		return m, nil
	}
	if m.overlayText != "" {
		if key == "b" {
			m.overlayText = ""
			m.overlayTitle = ""
			return m, nil
		}
		if key == "a" || key == "A" || key == "s" || key == "S" {
			cmd := m.jobAction(strings.ToLower(key))
			return m, cmd
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	switch m.State {
	case ViewDashboard:
		if key == "enter" {
			m.Query = strings.TrimSpace(m.input.Value())
			cmd := m.search(false)
			return m, cmd
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.Query = m.input.Value()
		return m, cmd
	case ViewResults, ViewHistory:
		length := len(m.Results)
		if m.State == ViewHistory {
			length = len(m.history)
		}
		switch key {
		case "up", "k":
			m.Cursor = max(0, m.Cursor-1)
		case "down", "j":
			m.Cursor = min(max(0, length-1), m.Cursor+1)
		case "home":
			m.Cursor = 0
		case "end":
			m.Cursor = max(0, length-1)
		case "enter":
			if job, ok := m.selectedJob(); ok {
				m.openOverlay(views.Tr(m.Preferences.Language, "Inspect"), views.Inspect(job, m.pulse, m.Preferences.ScamShield))
			}
		case "a", "A", "p", "P", "s", "S":
			cmd := m.jobAction(strings.ToLower(key))
			return m, cmd
		case "r", "R":
			if m.State == ViewHistory {
				cmd := m.loadHistory()
				return m, cmd
			}
			cmd := m.search(false)
			return m, cmd
		case "b", "B":
			return m.key(tea.KeyMsg{Type: tea.KeyEsc})
		}
	case ViewRadar:
		if key == "r" || key == "R" {
			cmd := m.search(true)
			return m, cmd
		}
		if key == "b" || key == "B" {
			return m.key(tea.KeyMsg{Type: tea.KeyEsc})
		}
	case ViewSettings:
		switch key {
		case "up", "k":
			m.settingsCursor = (m.settingsCursor + 2) % 3
		case "down", "j":
			m.settingsCursor = (m.settingsCursor + 1) % 3
		case "left", "h":
			m.changePreference(-1)
			cmd := m.savePreferences()
			return m, cmd
		case "right", "l", "enter", " ":
			m.changePreference(1)
			cmd := m.savePreferences()
			return m, cmd
		}
	case ViewDonate:
		if (key == "a" || key == "A") && m.sponsorURL != "" && !m.actionBusy {
			m.actionBusy = true
			services, url, ctx, nav := m.services, m.sponsorURL, m.ctx, m.navigationID
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
				defer cancel()
				return actionMsg{navigation: nav, action: "sponsor", err: services.OpenBrowser(ctx, url)}
			})
		}
	}
	return m, nil
}

func (m *Model) stopRequest() {
	if m.requestCancel != nil {
		m.requestCancel()
		m.requestCancel = nil
	}
	m.requestID++
	m.Loading = false
}
func (m Model) chooseMenu() (tea.Model, tea.Cmd) {
	m.stopRequest()
	m.navigationID++
	m.overlayText = ""
	m.overlayTitle = ""
	m.mainFocus = true
	m.Cursor = 0
	m.input.Blur()
	switch m.sidebarCursor {
	case 0:
		m.State = ViewDashboard
		m.category = ""
		cmd := m.input.Focus()
		return m, cmd
	case 1:
		m.State = ViewRadar
		cmd := m.search(true)
		return m, cmd
	case 2:
		m.category = "opportunities"
		m.Query = ""
		cmd := m.search(false)
		return m, cmd
	case 3:
		m.State = ViewHistory
		cmd := m.loadHistory()
		return m, cmd
	case 4:
		m.State = ViewSettings
	case 5:
		m.State = ViewDonate
	}
	return m, nil
}

func (m *Model) search(radar bool) tea.Cmd {
	if !m.ready || m.Engine == nil {
		m.StatusMessage = "Engine is not ready"
		return nil
	}
	m.stopRequest()
	m.navigationID++
	m.Loading = true
	m.overlayText = ""
	m.Cursor = 0
	if radar {
		m.State = ViewRadar
	} else {
		m.State = ViewResults
		m.Results = nil
	}
	m.input.Blur()
	m.StatusMessage = "Fetching listings..."
	ctx, cancel := context.WithTimeout(m.ctx, 60*time.Second)
	m.requestCancel = cancel
	id, engine, keyword, region, category := m.requestID, m.Engine, m.Query, m.Preferences.Region, m.category
	if region == "global" || region == "eu" || region == "asia" {
		region = ""
	}
	if radar {
		keyword = ""
		region = ""
		category = ""
	}
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		defer cancel()
		var jobs []models.JobListing
		var err error
		if category == "opportunities" {
			first, e1 := engine.FetchJobs(ctx, keyword, region, models.CategoryWeb3)
			second, e2 := engine.FetchJobs(ctx, keyword, region, models.CategoryBounty)
			jobs = append(first, second...)
			err = errors.Join(e1, e2)
		} else {
			jobs, err = engine.FetchJobs(ctx, keyword, region, "")
		}
		seen := map[string]bool{}
		unique := make([]models.JobListing, 0, len(jobs))
		for _, job := range jobs {
			key := job.Platform + ":" + job.ID
			if !seen[key] {
				seen[key] = true
				unique = append(unique, job)
			}
		}
		return fetchMsg{id: id, jobs: unique, pulse: engine.GetMarketPulse(""), err: err, radar: radar}
	})
}

func (m *Model) loadHistory() tea.Cmd {
	m.stopRequest()
	m.Loading = true
	m.history = nil
	id, read := m.requestID, m.services.History
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		entries, err := read()
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
		return historyMsg{id: id, entries: entries, err: err}
	})
}

func (m Model) selectedJob() (models.JobListing, bool) {
	if m.State == ViewHistory {
		if m.Cursor >= 0 && m.Cursor < len(m.history) {
			return m.history[m.Cursor].Job, true
		}
	} else if m.State == ViewResults {
		if m.Cursor >= 0 && m.Cursor < len(m.Results) {
			return m.Results[m.Cursor], true
		}
	}
	return models.JobListing{}, false
}

func (m *Model) jobAction(key string) tea.Cmd {
	job, ok := m.selectedJob()
	if !ok || m.actionBusy {
		return nil
	}
	if key == "a" {
		if job.IsMock {
			m.StatusMessage = "Demo listing: browser application disabled"
			return nil
		}
		if status, _ := security.ValidateJobListing(job); m.Preferences.ScamShield && status == models.RiskRed {
			m.StatusMessage = "ScamShield blocked this risky link. Enter: inspect flags."
			return nil
		}
		if err := client.ValidateURL(job.ApplyURL); err != nil {
			m.StatusMessage = err.Error()
			return nil
		}
	}
	m.actionBusy = true
	services, ctx, nav := m.services, m.ctx, m.navigationID
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		msg := actionMsg{navigation: nav}
		switch key {
		case "a":
			msg.action = "opened"
			ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			if err := services.OpenBrowser(ctx, job.ApplyURL); err != nil {
				msg.err = err
				return msg
			}
			if err := services.Log(job, "opened"); err != nil {
				msg.err = fmt.Errorf("browser opened; history failed: %w", err)
			}
		case "s":
			msg.action = "saved"
			msg.err = services.Log(job, "saved")
		case "p":
			msg.action = "pitch"
			msg.pitch = client.BuildPitch(job)
			if err := services.Log(job, "pitch"); err != nil {
				msg.err = fmt.Errorf("draft ready; history failed: %w", err)
			}
		}
		return msg
	})
}

func (m *Model) changePreference(direction int) {
	cycle := func(current string, values []string) string {
		index := 0
		for i, v := range values {
			if v == current {
				index = i
				break
			}
		}
		return values[(index+direction+len(values))%len(values)]
	}
	switch m.settingsCursor {
	case 0:
		m.Preferences.Language = cycle(m.Preferences.Language, []string{"en", "id", "jp"})
	case 1:
		m.Preferences.Region = cycle(m.Preferences.Region, []string{"global", "us", "eu", "asia", "id"})
	case 2:
		m.Preferences.ScamShield = !m.Preferences.ScamShield
	}
}
func (m *Model) savePreferences() tea.Cmd {
	if m.prefsSaving {
		m.prefsDirty = true
		return nil
	}
	m.prefsSaving = true
	prefs, save := m.Preferences, m.services.SavePreferences
	return func() tea.Msg { return preferencesMsg{err: save(prefs)} }
}

func filterRegion(jobs []models.JobListing, region string) []models.JobListing {
	if region != "eu" && region != "asia" {
		return jobs
	}
	key := "EU"
	if region == "asia" {
		key = "Asia"
	}
	out := make([]models.JobListing, 0, len(jobs))
	for _, job := range jobs {
		r := strings.ToLower(strings.TrimSpace(job.Region))
		if r == "worldwide" || r == "global" || r == "anywhere" || views.RegionalCounts([]models.JobListing{job})[key] > 0 {
			out = append(out, job)
		}
	}
	return out
}

func (m Model) contentWidth() int {
	if m.width >= 80 {
		return m.width - 24
	}
	return m.width
}
func (m Model) bodyHeight() int { return max(3, m.height-6) }
func (m *Model) resize() {
	m.input.Width = max(8, m.contentWidth()-6)
	m.viewport.Width = max(1, m.contentWidth()-4)
	m.viewport.Height = max(1, m.bodyHeight()-5)
	if m.overlayText != "" {
		m.wrapOverlay()
	}
}
func (m *Model) openOverlay(title, text string) {
	m.overlayTitle = title
	m.overlayText = text
	m.wrapOverlay()
	m.viewport.GotoTop()
}
func (m *Model) wrapOverlay() {
	lines := strings.Split(m.overlayText, "\n")
	for i, line := range lines {
		lines[i] = views.Wrap(line, m.viewport.Width)
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
}

func (m Model) View() string {
	if m.width < 50 || m.height < 16 {
		return views.Fit("HEXAJOBS.DEV\nResize terminal to at least 50 x 16.\nCtrl+C: quit", m.width, m.height)
	}
	lang := m.Preferences.Language
	status := views.Tr(lang, "Ready")
	if m.Loading {
		status = views.Tr(lang, "Loading")
	}
	if !m.ready && !m.Loading {
		status = "ERROR"
	}
	if m.demo {
		status = "DEMO · " + status
	}
	header := views.Toolbar(m.width, status, m.Preferences.Region, lang, m.styles)
	w, h := m.contentWidth(), m.bodyHeight()
	var main string
	if m.overlayText != "" {
		main = views.Box(m.styles.Title.Render(views.Line(m.overlayTitle, w-2))+"\n"+m.viewport.View()+"\n"+m.styles.Muted.Render("↑/↓ scroll · Esc close"), w, h, m.styles)
	} else {
		switch m.State {
		case ViewDashboard:
			main = views.Dashboard(m.input.View(), m.pulse, w, h, lang, m.styles)
		case ViewResults:
			main = views.Results(m.Results, m.Cursor, m.pulse, m.Preferences.ScamShield, w, h, lang, m.styles)
		case ViewRadar:
			main = views.Radar(m.radarJobs, m.pulse, w, h, lang, m.styles)
		case ViewSettings:
			main = views.Settings(lang, m.Preferences.Region, m.Preferences.ScamShield, m.settingsCursor, w, h, m.styles)
		case ViewDonate:
			main = views.Donate(m.sponsorURL, w, h, lang, m.styles)
		case ViewHistory:
			main = views.History(m.history, m.Cursor, w, h, lang, m.styles)
		}
	}
	body := main
	if m.width >= 80 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, views.Sidebar(m.sidebarCursor, !m.mainFocus, 23, h, lang, m.styles), " ", main)
	} else if !m.mainFocus {
		body = views.Sidebar(m.sidebarCursor, true, m.width, h, lang, m.styles)
	}
	footer := "Tab: menu/content  Enter: select  Esc: back  Ctrl+C: quit"
	if m.State == ViewResults || m.State == ViewHistory {
		footer = "[A] " + views.Tr(lang, "Apply") + "  [P] AI Pitch  [S] " + views.Tr(lang, "Save") + "  [B] " + views.Tr(lang, "Back")
	}
	busy := ""
	if m.Loading || m.actionBusy {
		busy = m.spinner.View() + " "
	}
	return views.Fit(header+"\n"+body+"\n"+m.styles.Muted.Render(views.Line(footer, m.width))+"\n"+busy+views.Line(m.StatusMessage, m.width-4)+"\n"+m.styles.Muted.Render("Tab: focus · ↑/↓ j/k: navigate · q: quit from main menu"), m.width, m.height)
}
