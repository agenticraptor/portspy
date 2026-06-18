// Package tui implements the interactive portspy terminal UI built on Bubble
// Tea: a live, filterable, sortable table of listening ports that you can kill
// with one key.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agenticraptor/portspy/internal/killer"
	"github.com/agenticraptor/portspy/internal/ports"
)

const refreshInterval = 3 * time.Second

// Run launches the interactive portspy TUI. The provided protocol filter is
// validated before the alternate screen is entered so errors surface cleanly.
func Run(_ context.Context, opts ports.Options) error {
	if err := ports.CheckProto(opts.Proto); err != nil {
		return err
	}
	p := tea.NewProgram(newModel(opts), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type model struct {
	opts   ports.Options
	killer *killer.Killer

	all  []ports.Listener
	view []ports.Listener

	table   table.Model
	cols    []table.Column
	filter  textinput.Model
	spinner spinner.Model
	help    help.Model
	keys    keyMap

	sortBy       ports.SortBy
	filtering    bool
	inspecting   bool
	confirming   bool
	pending      ports.Listener
	pendingForce bool

	status    string
	statusErr bool
	err       error
	loading   bool
	lastScan  time.Time

	width  int
	height int
}

func newModel(opts ports.Options) model {
	ti := textinput.New()
	ti.Placeholder = "filter by port, process, project, service…"
	ti.Prompt = "/"
	ti.CharLimit = 64

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = subtleStyle

	t := table.New(table.WithFocused(true))
	t.SetStyles(tableStyles())

	m := model{
		opts:    opts,
		killer:  killer.New(),
		sortBy:  ports.SortPort,
		filter:  ti,
		spinner: sp,
		table:   t,
		help:    help.New(),
		keys:    defaultKeys(),
		loading: true,
		width:   80,
		height:  24,
	}
	m.layout()
	return m
}

// --- messages & commands ---------------------------------------------------

type scannedMsg struct {
	listeners []ports.Listener
	err       error
}

type killedMsg struct {
	port    int
	pid     int
	outcome string
	err     error
}

type tickMsg struct{}

type clearStatusMsg struct{}

func scanCmd(opts ports.Options) tea.Cmd {
	return func() tea.Msg {
		ls, err := ports.Scan(opts)
		return scannedMsg{listeners: ls, err: err}
	}
}

func killCmd(k *killer.Killer, l ports.Listener, force bool) tea.Cmd {
	return func() tea.Msg {
		res, err := k.Kill(l.PID, force)
		return killedMsg{port: l.Port, pid: l.PID, outcome: res.Outcome.String(), err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func clearStatusCmd() tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

// --- bubbletea lifecycle ---------------------------------------------------

func (m model) Init() tea.Cmd {
	return tea.Batch(scanCmd(m.opts), m.spinner.Tick, tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case scannedMsg:
		m.loading = false
		m.lastScan = time.Now()
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.all = msg.listeners
		m.refreshView()
		return m, nil

	case killedMsg:
		if msg.err != nil {
			m.setStatus(fmt.Sprintf("failed to kill :%d (pid %d): %v", msg.port, msg.pid, msg.err), true)
		} else {
			m.setStatus(fmt.Sprintf(":%d (pid %d) %s", msg.port, msg.pid, msg.outcome), false)
		}
		m.loading = true
		return m, tea.Batch(scanCmd(m.opts), clearStatusCmd())

	case tickMsg:
		var cmds []tea.Cmd
		if !m.confirming {
			m.loading = true
			cmds = append(cmds, scanCmd(m.opts))
		}
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.confirming:
		switch {
		case key.Matches(msg, m.keys.Confirm):
			l, force := m.pending, m.pendingForce
			m.confirming = false
			m.pending = ports.Listener{}
			m.setStatus(fmt.Sprintf("killing :%d…", l.Port), false)
			m.loading = true
			return m, killCmd(m.killer, l, force)
		case key.Matches(msg, m.keys.Cancel):
			m.confirming = false
			m.pending = ports.Listener{}
			return m, nil
		}
		return m, nil

	case m.inspecting:
		switch {
		case key.Matches(msg, m.keys.Kill):
			m.inspecting = false
			cmd := m.askKill(false)
			return m, cmd
		case key.Matches(msg, m.keys.ForceKill):
			m.inspecting = false
			cmd := m.askKill(true)
			return m, cmd
		case key.Matches(msg, m.keys.Inspect), key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Quit):
			m.inspecting = false
			return m, nil
		}
		return m, nil

	case m.filtering:
		switch msg.String() {
		case "enter":
			m.filtering = false
			m.filter.Blur()
			return m, nil
		case "esc":
			m.filtering = false
			m.filter.Blur()
			m.filter.SetValue("")
			m.refreshView()
			return m, nil
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.refreshView()
		return m, cmd
	}

	// Normal mode.
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.layout()
		return m, nil
	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		m.filter.Focus()
		return m, textinput.Blink
	case key.Matches(msg, m.keys.Sort):
		m.sortBy = m.sortBy.Next()
		m.refreshView()
		m.setStatus("sorted by "+m.sortBy.String(), false)
		return m, clearStatusCmd()
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, scanCmd(m.opts)
	case key.Matches(msg, m.keys.Inspect):
		if _, ok := m.selected(); ok {
			m.inspecting = true
		}
		return m, nil
	case key.Matches(msg, m.keys.Kill):
		cmd := m.askKill(false)
		return m, cmd
	case key.Matches(msg, m.keys.ForceKill):
		cmd := m.askKill(true)
		return m, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *model) askKill(force bool) tea.Cmd {
	l, ok := m.selected()
	if !ok {
		return nil
	}
	if l.Self {
		m.setStatus("that's portspy itself — not killing it", true)
		return clearStatusCmd()
	}
	m.confirming = true
	m.pending = l
	m.pendingForce = force
	return nil
}

// --- view helpers ----------------------------------------------------------

func (m model) selected() (ports.Listener, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.view) {
		return ports.Listener{}, false
	}
	return m.view[i], true
}

func (m *model) setStatus(s string, isErr bool) {
	m.status = s
	m.statusErr = isErr
}

func (m *model) refreshView() {
	// Remember what's under the cursor so the selection follows the same
	// socket across refreshes/sorts rather than jumping to a random row.
	prev, hadPrev := m.selected()

	v := ports.Filter(m.all, m.filter.Value())
	ports.Sort(v, m.sortBy)
	m.view = v
	m.table.SetRows(m.rows())

	if hadPrev {
		for i, l := range v {
			if l.Port == prev.Port && l.Proto == prev.Proto && l.PID == prev.PID {
				m.table.SetCursor(i)
				return
			}
		}
	}
	if c := m.table.Cursor(); c >= len(v) {
		m.table.SetCursor(maxInt(0, len(v)-1))
	}
}

func (m *model) layout() {
	wPort, wProto, wPID, wProc, wUp, wAddr := 6, 5, 7, 18, 7, 16
	used := wPort + wProto + wPID + wProc + wUp + wAddr
	wWhat := m.width - used - 10
	if wWhat < 12 {
		wWhat = 12
	}
	m.cols = []table.Column{
		{Title: "PORT", Width: wPort},
		{Title: "PROTO", Width: wProto},
		{Title: "PID", Width: wPID},
		{Title: "PROCESS", Width: wProc},
		{Title: "UPTIME", Width: wUp},
		{Title: "ADDRESS", Width: wAddr},
		{Title: "WHAT", Width: wWhat},
	}
	m.table.SetColumns(m.cols)

	chrome := 6
	if m.help.ShowAll {
		chrome += 3
	}
	h := m.height - chrome
	if h < 3 {
		h = 3
	}
	m.table.SetHeight(h)
	m.help.Width = m.width
	m.table.SetRows(m.rows())
}

func (m model) rows() []table.Row {
	rows := make([]table.Row, 0, len(m.view))
	for _, l := range m.view {
		what := l.Label()
		if l.Self {
			what += " (self)"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", l.Port),
			string(l.Proto),
			fmt.Sprintf("%d", l.PID),
			clip(l.Process, colWidth(m.cols, 3)),
			l.HumanUptime(),
			clip(l.DisplayAddr(), colWidth(m.cols, 5)),
			clip(what, colWidth(m.cols, 6)),
		})
	}
	return rows
}

func (m model) View() string {
	if m.confirming {
		return m.confirmView()
	}
	if m.inspecting {
		return m.inspectView()
	}
	return m.titleView() + "\n\n" +
		m.table.View() + "\n" +
		m.statusView() + "\n" +
		m.help.View(m.keys)
}

func (m model) titleView() string {
	left := titleStyle.Render("🔌 portspy")
	meta := fmt.Sprintf("  %d listening  ·  sort: %s", len(m.view), m.sortBy)
	if f := m.filter.Value(); f != "" {
		meta += fmt.Sprintf("  ·  filter: %q", f)
	}
	left += subtleStyle.Render(meta)

	var right string
	switch {
	case m.loading:
		right = m.spinner.View() + "scanning"
	case !m.lastScan.IsZero():
		right = "updated " + m.lastScan.Format("15:04:05")
	}
	right = subtleStyle.Render(right)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m model) statusView() string {
	switch {
	case m.filtering:
		return m.filter.View()
	case m.err != nil:
		return errStyle.Render("error: " + m.err.Error())
	case m.status != "":
		if m.statusErr {
			return errStyle.Render(m.status)
		}
		return okStyle.Render(m.status)
	case len(m.view) == 0:
		return subtleStyle.Render("no matching listeners")
	}
	if l, ok := m.selected(); ok {
		ctx := l.Command
		if ctx == "" {
			ctx = l.Exe
		}
		if ctx == "" {
			ctx = l.Process
		}
		return subtleStyle.Render("▸ " + clip(ctx, maxInt(0, m.width-2)))
	}
	return ""
}

func (m model) confirmView() string {
	l := m.pending
	verb := "Terminate"
	if m.pendingForce {
		verb = "Force-kill"
	}

	lines := []string{
		boxTitleStyle.Render(verb + " this process?"),
		"",
		fmt.Sprintf("Port      :%d/%s", l.Port, l.Proto),
		fmt.Sprintf("Process   %s (pid %d)", l.Process, l.PID),
		fmt.Sprintf("Uptime    %s", l.HumanUptime()),
	}
	if !l.Project.Empty() && l.Project.Name != "" {
		lines = append(lines, fmt.Sprintf("Project   %s", l.Project.Name))
	}
	if l.Service != "" {
		lines = append(lines, fmt.Sprintf("Service   %s", l.Service))
	}
	if chain := lineageString(l.Parents); chain != "" {
		lines = append(lines, fmt.Sprintf("Parent    %s", clip(chain, 50)))
	}
	if l.Command != "" {
		lines = append(lines, "", subtleStyle.Render(clip(l.Command, 54)))
	}
	if l.Exposed {
		lines = append(lines, "", warnStyle.Render("⚠ bound beyond localhost (exposed)"))
	}
	lines = append(lines,
		"",
		okStyle.Render("[y] yes")+"    "+errStyle.Render("[n] cancel"),
	)

	box := boxStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// inspectView renders a full-detail card for the selected listener, including
// the process lineage, executable, project root, and exact start time.
func (m model) inspectView() string {
	l, ok := m.selected()
	if !ok {
		return ""
	}

	addr := l.DisplayAddr()
	if l.Exposed {
		addr += "  (exposed beyond localhost)"
	}

	lines := []string{
		boxTitleStyle.Render(fmt.Sprintf(":%d/%s", l.Port, l.Proto)),
		"",
		fmt.Sprintf("Process   %s (pid %d)", l.Process, l.PID),
	}
	if l.Service != "" {
		lines = append(lines, fmt.Sprintf("Service   %s", l.Service))
	}
	if !l.Project.Empty() {
		proj := l.Project.Name
		if l.Project.Type != "" {
			proj += fmt.Sprintf(" (%s)", l.Project.Type)
		}
		lines = append(lines, fmt.Sprintf("Project   %s", proj))
		if l.Project.Root != "" {
			lines = append(lines, fmt.Sprintf("Root      %s", clip(l.Project.Root, 56)))
		}
	}
	lines = append(lines, fmt.Sprintf("Address   %s", addr))
	if !l.CreateTime.IsZero() {
		lines = append(lines, fmt.Sprintf("Started   %s  (%s ago)", l.CreateTime.Format("2006-01-02 15:04:05"), l.HumanUptime()))
	}
	if chain := lineageString(l.Parents); chain != "" {
		lines = append(lines, fmt.Sprintf("Parent    %s", clip(chain, 56)))
	}
	if l.Exe != "" {
		lines = append(lines, fmt.Sprintf("Exe       %s", clip(l.Exe, 56)))
	}
	if l.Command != "" {
		lines = append(lines, "", subtleStyle.Render(clip(l.Command, 60)))
	}
	lines = append(lines,
		"",
		subtleStyle.Render("[x] kill   [X] force-kill   [enter/esc] close"),
	)

	box := boxStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// lineageString renders a parent chain as "npm ← zsh ← login", newest first.
func lineageString(parents []ports.ProcRef) string {
	if len(parents) == 0 {
		return ""
	}
	names := make([]string, 0, len(parents))
	for _, p := range parents {
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("pid %d", p.PID)
		}
		names = append(names, name)
	}
	return strings.Join(names, " ← ")
}

// --- small utilities -------------------------------------------------------

func colWidth(cols []table.Column, i int) int {
	if i < 0 || i >= len(cols) {
		return 12
	}
	return cols[i].Width
}

func clip(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
