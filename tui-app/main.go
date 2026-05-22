package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"axis-mundi-tui/mcp"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultBaseURL    = "http://localhost:8080/mcp"
	telemetrySockPath = "/tmp/foundations.sock"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#56F47D")).
			Padding(0, 1)

	stateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#F4A056")).
			Padding(0, 1).
			Bold(true)

	stateLogStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F4A056")).Bold(true)
	taskLogStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#56F4F4")).Italic(true)
	logStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	activeBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	inactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	docStyle = lipgloss.NewStyle().Margin(1, 1)

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

const (
	focusList = iota
	focusLogs
)

type TelemetryEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	State     string    `json:"state,omitempty"`
	AgentID   string    `json:"agent_id"`
}

type item struct {
	id, title, desc, itemType, status string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return fmt.Sprintf("[%s] %s (%s)", i.itemType, i.desc, i.status) }
func (i item) FilterValue() string { return i.title }

type model struct {
	mcpClient      *mcp.Client
	list           list.Model
	viewport       viewport.Model
	logs           []string
	agentID        string
	agentState     string
	err            error
	width, height  int
	pollingEnabled bool
	focus          int
}

type listMsg []mcp.WorkspaceItem
type telemetryMsg TelemetryEvent
type errorMsg error
type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchList,
		m.tick(),
	)
}

func (m model) tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) fetchList() tea.Msg {
	items, err := m.mcpClient.ListWorkspace()
	if err != nil {
		return errorMsg(err)
	}
	return listMsg(items)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		h, v := docStyle.GetFrameSize()
		
		listWidth := m.width / 3
		logWidth := m.width - listWidth - h - 4
		
		m.list.SetSize(listWidth-h, msg.Height-v-5)
		m.viewport.Width = logWidth
		m.viewport.Height = msg.Height - v - 5

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.focus == focusList {
				m.focus = focusLogs
			} else {
				m.focus = focusList
			}
		case "p":
			m.pollingEnabled = !m.pollingEnabled
		}

	case listMsg:
		var items []list.Item
		for _, mi := range msg {
			items = append(items, item{
				id:       mi.ID,
				title:    mi.Title,
				desc:     mi.Snippet,
				itemType: mi.Type,
				status:   mi.Status,
			})
		}
		m.list.SetItems(items)

	case telemetryMsg:
		if msg.AgentID != "" {
			m.agentID = msg.AgentID
		}
		if msg.State != "" {
			m.agentState = msg.State
		}

		timestamp := msg.Timestamp.Format("15:04:05")
		
		var typeStyle lipgloss.Style
		var msgStyle lipgloss.Style
		
		switch msg.Type {
		case "STATE":
			typeStyle = stateStyle
			msgStyle = stateLogStyle
		case "TASK":
			typeStyle = statusStyle.Copy().Background(lipgloss.Color("#56F4F4"))
			msgStyle = taskLogStyle
		default:
			typeStyle = statusStyle
			msgStyle = logStyle
		}

		logLine := fmt.Sprintf("%s %s %s", dimStyle.Render(timestamp), typeStyle.Render(msg.Type), msgStyle.Render(msg.Message))
		m.logs = append(m.logs, logLine)
		if len(m.logs) > 500 {
			m.logs = m.logs[1:]
		}

		content := ""
		for _, l := range m.logs {
			content += l + "\n"
		}
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()

	case tickMsg:
		if m.pollingEnabled {
			cmds = append(cmds, m.fetchList)
		}
		cmds = append(cmds, m.tick())

	case errorMsg:
		// Don't treat telemetry connection errors as fatal for the UI
		m.logs = append(m.logs, dimStyle.Render(fmt.Sprintf("System Error: %v", msg)))
		m.viewport.SetContent(fmt.Sprintf("%s\n%v", m.viewport.View(), msg))
	}

	// Route updates based on focus
	if m.focus == focusList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		titleStyle.Render("Foundations Telemetry"),
		" ",
		stateStyle.Render(fmt.Sprintf("Agent: %s [%s]", m.agentID, m.agentState)),
	)

	if m.pollingEnabled {
		header += " " + statusStyle.Render("Polling: ON")
	}

	listStyle := inactiveBorder
	viewportStyle := inactiveBorder

	if m.focus == focusList {
		listStyle = activeBorder
	} else {
		viewportStyle = activeBorder
	}

	listWidth := m.width / 3
	logWidth := m.width - listWidth - 4

	leftPanel := listStyle.Width(listWidth).Height(m.height - 7).Render(m.list.View())
	rightPanel := viewportStyle.Width(logWidth).Height(m.height - 7).Render(m.viewport.View())

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	footer := "q: quit | tab: switch focus | p: toggle polling | enter: select (list)"

	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, mainContent, footer))
}

func main() {
	baseURL := os.Getenv("AXIS_MUNDI_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	items := []list.Item{}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Workspace Items"

	m := model{
		mcpClient:      mcp.NewClient(baseURL),
		list:           l,
		viewport:       viewport.New(0, 0),
		pollingEnabled: true,
		agentID:        "Unknown",
		agentState:     "DISCONNECTED",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start telemetry subscriber
	go func() {
		for {
			conn, err := net.Dial("unix", telemetrySockPath)
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}

			scanner := bufio.NewScanner(conn)
			for scanner.Scan() {
				var event TelemetryEvent
				if err := json.Unmarshal(scanner.Bytes(), &event); err == nil {
					p.Send(telemetryMsg(event))
				}
			}
			conn.Close()
			time.Sleep(1 * time.Second)
		}
	}()

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
