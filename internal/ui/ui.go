package ui

import (
	"cpa-quota/internal/config"
	"cpa-quota/internal/cpa"
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"
)

type tickMsg time.Time

type uiMode int

const (
	modeQuota uiMode = iota
	modeConfig
)

type snapshotMsg struct {
	seq      int
	snapshot cpa.Snapshot
	err      error
}

type versionMsg struct {
	seq    int
	latest string
	err    error
}

type configuredMsg struct {
	config   config.Config
	snapshot cpa.Snapshot
	err      error
}

type uiModel struct {
	configPath string
	config     config.Config
	hasConfig  bool
	mode       uiMode


	baseInput textinput.Model
	keyInput  textinput.Model
	focused   int

	viewport viewport.Model
	snapshot cpa.Snapshot
	fetching bool
	saving   bool
	errText  string

	refreshSeq      int
	versionChecking bool
	latestVersion   string
	latestErr       string

	width  int
	height int
}

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	groupStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	accountStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warningStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	goodStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	borderStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	configLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
)

func newUIModel(path string) uiModel {
	cfg, err := config.Load(path)
	hasConfig := err == nil
	if errors.Is(err, config.ErrConfigNotFound) {
		cfg = config.DefaultConfig()
	}

	baseInput := textinput.New()
	baseInput.Prompt = ""
	baseInput.Placeholder = config.DefaultConfig().BaseURL
	baseInput.SetValue(cfg.BaseURL)
	baseInput.CharLimit = 2048

	keyInput := textinput.New()
	keyInput.Prompt = ""
	keyInput.Placeholder = "Management key"
	keyInput.SetValue(cfg.ManagementKey)
	keyInput.CharLimit = 4096
	keyInput.EchoMode = textinput.EchoPassword
	keyInput.EchoCharacter = '•'

	model := uiModel{
		configPath: path,
		config:     cfg,
		hasConfig:  hasConfig,
		mode:       modeQuota,
		baseInput:  baseInput,
		keyInput:   keyInput,
		viewport:   viewport.New(0, 0),
		fetching:   hasConfig,
		versionChecking: hasConfig,
		refreshSeq: seqForInitialFetch(hasConfig),
	}
	if !hasConfig {
		model.mode = modeConfig
		model.focusConfigField(0)
		if err != nil && !errors.Is(err, config.ErrConfigNotFound) {
			model.errText = err.Error()
		}
	}
	return model
}

func (m uiModel) Init() tea.Cmd {
	if m.hasConfig {
		return tea.Batch(fetchSnapshotCmd(m.config, m.refreshSeq), fetchVersionCmd(m.config, m.refreshSeq), countdownTick())
	}
	return countdownTick()
}

func seqForInitialFetch(hasConfig bool) int {
	if hasConfig {
		return 1
	}
	return 0
}

func countdownTick() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m uiModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tickMsg:
		if m.mode == modeQuota && !m.snapshot.FetchedAt.IsZero() {
			m.refreshViewport()
		}
		return m, countdownTick()
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		m.resize()
		return m, nil
	case snapshotMsg:
		if message.seq != 0 && message.seq != m.refreshSeq {
			return m, nil
		}
		m.fetching = false
		if message.err != nil {
			m.errText = message.err.Error()
			return m, nil
		}
		m.errText = ""
		m.snapshot = message.snapshot
		m.refreshViewport()
		return m, nil
	case versionMsg:
		if message.seq != 0 && message.seq != m.refreshSeq {
			return m, nil
		}
		m.versionChecking = false
		if message.err != nil {
			m.latestErr = message.err.Error()
			m.latestVersion = ""
			return m, nil
		}
		m.latestErr = ""
		m.latestVersion = message.latest
		return m, nil
	case configuredMsg:
		m.saving = false
		if message.err != nil {
			m.errText = message.err.Error()
			return m, nil
		}
		m.config = message.config
		m.hasConfig = true
		m.snapshot = message.snapshot
		m.errText = ""
		m.latestVersion = ""
		m.latestErr = ""
		m.versionChecking = true
		m.mode = modeQuota
		m.refreshViewport()
		return m, tea.Batch(fetchVersionCmd(m.config, m.nextSeq()))
	case tea.KeyMsg:
		if m.mode == modeConfig {
			return m.updateConfig(message)
		}
		return m.updateQuota(message)
	}

	if m.mode == modeConfig {
		return m.updateFocusedInput(message)
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(message)
	return m, cmd
}

func (m uiModel) updateQuota(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "esc":
		return m, tea.Quit
	case "r", "R":
		if m.fetching {
			return m, nil
		}
		m.fetching = true
		m.versionChecking = true
		m.errText = ""
		seq := m.nextSeq()
		return m, tea.Batch(fetchSnapshotCmd(m.config, seq), fetchVersionCmd(m.config, seq))
	case "c", "C":
		if m.fetching {
			return m, nil
		}
		m.mode = modeConfig
		m.errText = ""
		m.baseInput.SetValue(m.config.BaseURL)
		m.keyInput.SetValue(m.config.ManagementKey)
		m.focusConfigField(0)
		return m, nil
	case "g":
		m.viewport.GotoTop()
		return m, nil
	case "G":
		m.viewport.GotoBottom()
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(key)
	return m, cmd
}

func (m uiModel) updateConfig(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.String() == "esc" {
		if m.hasConfig {
			m.mode = modeQuota
			m.errText = ""
			return m, nil
		}
		return m, tea.Quit
	}
	if m.saving {
		return m, nil
	}
	switch key.String() {
	case "tab", "shift+tab", "up", "down":
		m.focusConfigField(1 - m.focused)
		return m, nil
	case "enter":
		if m.focused == 0 {
			m.focusConfigField(1)
			return m, nil
		}
		return m.submitConfig()
	case "ctrl+s":
		return m.submitConfig()
	}
	return m.updateFocusedInput(key)
}

func (m uiModel) updateFocusedInput(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.focused == 0 {
		m.baseInput, cmd = m.baseInput.Update(message)
	} else {
		m.keyInput, cmd = m.keyInput.Update(message)
	}
	return m, cmd
}

func (m uiModel) submitConfig() (tea.Model, tea.Cmd) {
	cfg := config.Config{BaseURL: m.baseInput.Value(), ManagementKey: m.keyInput.Value()}.Normalized()
	if err := cfg.Validate(); err != nil {
		return m, nil
	}
	m.saving = true
	m.errText = ""
	return m, configureCmd(m.configPath, cfg)
}

func (m *uiModel) focusConfigField(index int) {
	m.focused = index
	if index == 0 {
		m.baseInput.Focus()
		m.keyInput.Blur()
		return
	}
	m.baseInput.Blur()
	m.keyInput.Focus()
}

func (m *uiModel) resize() {
	width := max(20, m.width-4)
	m.baseInput.Width = max(10, width-2)
	m.keyInput.Width = max(10, width-2)
	m.viewport.Width = max(1, m.width)
	m.viewport.Height = max(1, m.height-4)
	m.refreshViewport()
}

func (m *uiModel) refreshViewport() {
	if m.fetching && m.snapshot.FetchedAt.IsZero() {
		m.viewport.SetContent("\n" + warningStyle.Render("Loading quotas…"))
		return
	}
	m.viewport.SetContent(renderSnapshot(m.snapshot, m.viewport.Width, time.Now()))
}

func (m uiModel) View() string {
	if m.mode == modeConfig {
		return m.configView()
	}

	status := ""
	switch {
	case m.fetching:
		status = warningStyle.Render("Refreshing…")
	case !m.snapshot.FetchedAt.IsZero() && m.width >= 65:
		status = dimStyle.Render("Updated " + m.snapshot.FetchedAt.Local().Format("02/01 15:04:05"))
	}
	versionLine := versionStatusLine(m, m.versionChecking)
	header := titleStyle.Render("CPA Quota") + "\n" + versionLine
	if status != "" {
		header += "  " + status
	}

	footerText := "R refresh  C configure  j/k scroll  g/G top/bottom  q/Esc close"
	switch {
	case m.width > 0 && m.width < 32:
		footerText = "R · C · q"
	case m.width > 0 && m.width < 63:
		footerText = "R refresh · C config · q close"
	}
	footer := dimStyle.Render(footerText)
	if m.errText != "" {
		footer = errorStyle.Render("Unavailable: "+compactUIError(m.errText)) + "  " + footer
	}
	return header + "\n" + m.viewport.View() + "\n" + footer
}

func (m uiModel) configView() string {
	width := max(24, m.width-4)
	content := []string{
		titleStyle.Render("Configure CPA Quota"),
		"",
		configLabelStyle.Render("CLIProxyAPI Base URL"),
		m.baseInput.View(),
		"",
		configLabelStyle.Render("Management key"),
		m.keyInput.View(),
	}
	if m.errText != "" {
		content = append(content, "", errorStyle.Render(compactUIError(m.errText)))
	}
	state := "Enter save"
	if m.saving {
		state = "Checking connection…"
	}
	content = append(content, "", dimStyle.Render(state+"  Tab switch field  Esc cancel/close"))
	return borderStyle.Width(width).Render(strings.Join(content, "\n"))
}

func versionStatusLine(model uiModel, checking bool) string {
	current := strings.TrimSpace(model.snapshot.CurrentVersion)
	latest := strings.TrimSpace(model.latestVersion)
	latestFailed := strings.TrimSpace(model.latestErr) != ""

	switch {
	case checking && current == "":
		return dimStyle.Render("CPA version · checking latest…")
	case checking:
		return dimStyle.Render("CPA " + current + " · checking latest…")
	case current == "" && latestFailed:
		return dimStyle.Render("CPA version unknown · latest check unavailable")
	case current == "" && latest != "":
		return dimStyle.Render("CPA version unknown · latest " + latest)
	case current == "":
		return dimStyle.Render("CPA version unknown")
	case latestFailed:
		return dimStyle.Render("CPA " + current + " · latest check unavailable")
	case latest == "":
		return dimStyle.Render("CPA " + current)
	}
	comparison, ok := cpa.CompareVersions(current, latest)
	if !ok {
		return dimStyle.Render("CPA " + current + " · latest " + latest)
	}
	switch comparison {
	case 0:
		return dimStyle.Render("CPA " + current)
	case -1:
		if model.width > 0 && model.width < 32 {
			return warningStyle.Render(current + " → " + latest + " avail")
		}
		return warningStyle.Render("CPA " + current + " → " + latest + " available")
	default:
		return dimStyle.Render("CPA " + current + " · ahead of latest " + latest)
	}
}

func (m *uiModel) nextSeq() int {
	m.refreshSeq++
	return m.refreshSeq
}

func fetchSnapshotCmd(cfg config.Config, seq int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		snapshot, err := cpa.NewClient(cfg).FetchSnapshot(ctx)
		return snapshotMsg{seq: seq, snapshot: snapshot, err: err}
	}
}

func fetchVersionCmd(cfg config.Config, seq int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		latest, err := cpa.NewClient(cfg).FetchLatestVersion(ctx)
		return versionMsg{seq: seq, latest: latest, err: err}
	}
}

func configureCmd(path string, cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		snapshot, err := cpa.NewClient(cfg).FetchSnapshot(ctx)
		if err == nil {
			err = config.Save(path, cfg)
		}
		return configuredMsg{config: cfg, snapshot: snapshot, err: err}
	}
}

func renderSnapshot(snapshot cpa.Snapshot, width int, now time.Time) string {
	if len(snapshot.Groups) == 0 {
		return "\n" + dimStyle.Render("No Codex, Antigravity, or Claude OAuth accounts were returned by CPA.")
	}
	if width <= 0 {
		width = 80
	}
	var sections []string
	for _, group := range snapshot.Groups {
		shortenLabels := !groupFitsFullLabels(group, width, now)
		labelWidth := groupLabelWidth(group, shortenLabels, 0)
		maxAllowed := width - 4 - 2 - 4 - 2 - 10
		if maxAllowed < 1 {
			maxAllowed = 1
		}
		if labelWidth > maxAllowed {
			labelWidth = maxAllowed
		}
		barWidth := calculateBarWidth(width, labelWidth)
		lines := []string{groupStyle.Render(group.Title)}
		for _, account := range group.Accounts {
			lines = append(lines, "  "+accountStyle.Render(account.Name)+renderAccountBadges(account))
			for _, window := range account.Windows {
				lines = append(lines, renderQuotaLine(window, now, labelWidth, barWidth, width, shortenLabels))
			}
			if account.Provider == cpa.ProviderCodex {
				resets := "—"
				if account.ManualResets != nil {
					resets = fmt.Sprintf("%d", *account.ManualResets)
				}
				lines = append(lines, "    Manual resets  "+resets)
			}
			if account.Error != "" {
				lines = append(lines, "    "+errorStyle.Render("Unavailable: "+compactUIError(account.Error)))
			}
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return strings.Join(sections, "\n\n")
}

func compactLabel(raw string) string {
	switch raw {
	case "Claude & GPT models":
		return "Claude/GPT"
	case "Gemini models":
		return "Gemini"
	default:
		return raw
	}
}

func resolveLabel(raw string, shorten bool, maxLen int) string {
	label := raw
	if shorten {
		label = compactLabel(raw)
	}
	if maxLen > 0 && len(label) > maxLen {
		if maxLen <= 1 {
			return "…"
		}
		return label[:maxLen-1] + "…"
	}
	return label
}

func groupLabelWidth(group cpa.ProviderQuota, shorten bool, maxLen int) int {
	width := 0
	for _, account := range group.Accounts {
		for _, window := range account.Windows {
			l := uniseg.StringWidth(resolveLabel(window.Label, shorten, maxLen))
			if l > width {
				width = l
			}
		}
	}
	return width
}

func groupFitsFullLabels(group cpa.ProviderQuota, width int, now time.Time) bool {
	if width < 55 {
		return false
	}
	fullLabelWidth := groupLabelWidth(group, false, 0)
	for _, account := range group.Accounts {
		for _, window := range account.Windows {
			minLineLen := 4 + fullLabelWidth + 2 + 4 + 2 + uniseg.StringWidth(resetCountdownText(window, now))
			if minLineLen > width {
				return false
			}
		}
	}
	return true
}

func calculateBarWidth(totalWidth int, labelWidth int) int {
	available := totalWidth - (4 + labelWidth + 2)
	if available < 6 {
		return 6
	}
	if available > 24 {
		return 24
	}
	return available
}

func renderQuotaLine(window cpa.QuotaWindow, now time.Time, labelWidth int, barWidth int, totalWidth int, shortenLabels bool) string {
	indent := "    "
	label := resolveLabel(window.Label, shortenLabels, labelWidth)
	paddedLabel := fmt.Sprintf("%-*s", labelWidth, label)
	meta := indent + paddedLabel + "  "
	barPad := indent + strings.Repeat(" ", labelWidth+2)

	resetStr := resetCountdownText(window, now)
	timestamp := ""
	if window.ResetAt != nil {
		timestamp = window.ResetAt.Local().Format("02/01 15:04")
	}

	var pctStr string
	var barStyle lipgloss.Style
	var remaining float64
	if window.Remaining == nil {
		pctStr = dimStyle.Render("—")
		barStyle = dimStyle
	} else {
		remaining = clamp(*window.Remaining, 0, 100)
		barStyle = healthStyle(remaining)
		pctStr = barStyle.Render(fmt.Sprintf("%3.0f%%", remaining))
	}

	countdownRendered := resetStr
	if window.ResetAt == nil {
		countdownRendered = dimStyle.Render("reset —")
	}

	meta += pctStr + "  " + countdownRendered
	if timestamp != "" && totalWidth >= 50 {
		withTimestampLen := 4 + labelWidth + 2 + 4 + 2 + uniseg.StringWidth(resetStr) + 2 + 11
		if withTimestampLen <= totalWidth {
			meta += "  " + dimStyle.Render(timestamp)
		}
	}

	var bar string
	if window.Remaining == nil {
		bar = dimStyle.Render(strings.Repeat("─", barWidth))
	} else {
		filled := int(math.Round(remaining / 100 * float64(barWidth)))
		bar = barStyle.Render(strings.Repeat("━", filled)) + dimStyle.Render(strings.Repeat("─", barWidth-filled))
	}
	return meta + "\n" + barPad + bar
}

func resetCountdownText(window cpa.QuotaWindow, now time.Time) string {
	if window.ResetAt == nil {
		return "reset —"
	}
	return resetCountdown(*window.ResetAt, now)
}

func healthStyle(remaining float64) lipgloss.Style {
	switch {
	case remaining <= 20:
		return errorStyle
	case remaining <= 50:
		return warningStyle
	default:
		return goodStyle
	}
}

func resetCountdown(target, now time.Time) string {
	until := target.Sub(now)
	if until <= 0 {
		return "ready"
	}
	if until < time.Minute {
		return "in <1m"
	}
	days := int(until.Hours()) / 24
	hours := int(until.Hours()) % 24
	minutes := int(until.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("in %dd%dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("in %dh%dm", hours, minutes)
	default:
		return fmt.Sprintf("in %dm", minutes)
	}
}

func renderAccountBadges(account cpa.AccountQuota) string {
	var badges []string
	if account.Disabled {
		badges = append(badges, warningStyle.Render("disabled"))
	}
	if account.Unavailable {
		badges = append(badges, errorStyle.Render("unavailable"))
	}
	status := strings.ToLower(strings.TrimSpace(account.Status))
	if status != "" && status != "ready" && status != "ok" {
		badges = append(badges, dimStyle.Render(status))
	}
	if len(badges) == 0 {
		return ""
	}
	return "  [" + strings.Join(badges, ", ") + "]"
}

func compactUIError(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 140 {
		return message[:137] + "..."
	}
	return message
}

// NewModel initializes the Bubble Tea model for Quota View.
func NewModel(configPath string) tea.Model {
	return newUIModel(configPath)
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Min(maximum, math.Max(minimum, value))
}
