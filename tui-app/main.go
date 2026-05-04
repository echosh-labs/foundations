package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"axis-mundi-tui/mcp"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultBaseURL = "http://localhost:8080/mcp"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#56F47D")).
			Padding(0, 1)

	docStyle = lipgloss.NewStyle().Margin(1, 2)

	itemStyle = lipgloss.NewStyle().PaddingLeft(4)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170"))

	detailStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1)
)

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
	selectedItem   *mcp.WorkspaceItem
	itemContent    string
	itemStatus     string
	err            error
	width, height  int
	pollingEnabled bool
}

type listMsg []mcp.WorkspaceItem
type contentMsg struct {
	content string
	status  string
}
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

func (m model) fetchContent(id, itemType string) tea.Cmd {
	return func() tea.Msg {
		content, err := m.mcpClient.GetItemContent(id, itemType)
		if err != nil {
			return errorMsg(err)
		}
		status, err := m.mcpClient.GetStatus(id)
		if err != nil {
			return errorMsg(err)
		}
		return contentMsg{content: content, status: status}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width/3-h, msg.Height-v-3)
		m.viewport.Width = msg.Width*2/3 - h - 4
		m.viewport.Height = msg.Height - v - 5
		m.viewport.SetContent(m.itemContent)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(item); ok {
				m.itemContent = "Loading..."
				m.viewport.SetContent(m.itemContent)
				cmds = append(cmds, m.fetchContent(i.id, i.itemType))
			}
		case "p":
			m.pollingEnabled = !m.pollingEnabled
		case "s":
			// Handle status update (simplified for now)
			if i, ok := m.list.SelectedItem().(item); ok {
				var newStatus string
				switch i.status {
				case "Pending":
					newStatus = "Execute"
				case "Execute":
					newStatus = "Active"
				case "Active":
					newStatus = "Complete"
				case "Complete":
					newStatus = "Pending"
				default:
					newStatus = "Pending"
				}
				err := m.mcpClient.SetStatus(i.id, newStatus)
				if err != nil {
					m.err = err
				} else {
					cmds = append(cmds, m.fetchList)
				}
			}
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

	case contentMsg:
		m.itemContent = msg.content
		m.itemStatus = msg.status
		m.viewport.SetContent(fmt.Sprintf("Status: %s\n\n%s", m.itemStatus, m.itemContent))

	case tickMsg:
		if m.pollingEnabled {
			cmds = append(cmds, m.fetchList)
		}
		cmds = append(cmds, m.tick())

	case errorMsg:
		m.err = msg
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}

	header := titleStyle.Render("Axis-Mundi TUI")
	if m.pollingEnabled {
		header += " " + statusStyle.Render("Polling: ON")
	} else {
		header += " " + statusStyle.Render("Polling: OFF")
	}

	leftPane := m.list.View()
	rightPane := detailStyle.Render(m.viewport.View())

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	footer := "q: quit | enter: view details | s: cycle status | p: toggle polling"
	
	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, content, footer))
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
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
