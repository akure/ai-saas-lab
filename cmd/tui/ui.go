package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aisaaslab/internal/modules/completion"
)

type TabIndex int

const (
	TabDashboard TabIndex = iota
	TabChat
	TabSessions
	TabAdmin
)

// Custom Bubbletea Messages
type TokenMsg string
type StreamDoneMsg struct{}
type StreamErrMsg struct{ err error }
type UsageLoadedMsg struct{ tokens int }
type PersonasLoadedMsg struct{ personas []completion.Persona }
type SessionsLoadedMsg struct{ sessions []*completion.Session }
type KeyCreatedMsg struct{ key, plan string }

type Model struct {
	client          *APIClient
	activeTab       TabIndex
	width           int
	height          int
	apiKey          string
	plan            string
	dailyQuota      int
	tokensUsed      int
	activeSessionID string
	selectedPersona string
	includeMemory   bool
	personas        []completion.Persona
	sessions        []*completion.Session
	selectedSessIdx int
	chatHistory     []completion.ChatMessage
	streamingText   strings.Builder
	isStreaming     bool
	cancelStream    context.CancelFunc
	tokenChan       chan string
	errChan         chan error
	textarea        textarea.Model
	viewport        viewport.Model
	statusMsg       string
}

func NewModel(client *APIClient, initialKey string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your prompt... (Press Enter to send)"
	ta.Focus()
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 18)

	return Model{
		client:          client,
		activeTab:       TabDashboard,
		apiKey:          initialKey,
		plan:            "pro",
		dailyQuota:      50,
		selectedPersona: "developer",
		includeMemory:   true,
		textarea:        ta,
		viewport:        vp,
		tokenChan:       make(chan string, 100),
		errChan:         make(chan error, 10),
		statusMsg:       "Connected to AI SaaS Lab server",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.cmdFetchUsage(),
		m.cmdFetchPersonas(),
		m.cmdFetchSessions(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.cancelStream != nil {
				m.cancelStream()
			}
			return m, tea.Quit

		case "tab":
			m.activeTab = (m.activeTab + 1) % 4
			return m, nil

		case "1":
			m.activeTab = TabDashboard
			return m, nil
		case "2":
			m.activeTab = TabChat
			return m, nil
		case "3":
			m.activeTab = TabSessions
			return m, m.cmdFetchSessions()
		case "4":
			m.activeTab = TabAdmin
			return m, nil
		}

		// Tab-specific keybindings
		switch m.activeTab {
		case TabChat:
			switch msg.String() {
			case "ctrl+r":
				m.cyclePersona()
				m.statusMsg = "Switched Persona: " + m.selectedPersona
				return m, nil
			case "ctrl+m":
				m.includeMemory = !m.includeMemory
				m.statusMsg = fmt.Sprintf("Cross-Session Memory Recall: %v", m.includeMemory)
				return m, nil
			case "ctrl+l":
				m.chatHistory = nil
				m.viewport.SetContent("")
				m.statusMsg = "Cleared chat window"
				return m, nil
			case "enter":
				if !m.isStreaming {
					prompt := strings.TrimSpace(m.textarea.Value())
					if prompt != "" {
						m.textarea.Reset()
						return m, m.startStreaming(prompt)
					}
				}
				return m, nil
			}

		case TabSessions:
			switch msg.String() {
			case "up", "k":
				if m.selectedSessIdx > 0 {
					m.selectedSessIdx--
				}
				return m, nil
			case "down", "j":
				if m.selectedSessIdx < len(m.sessions)-1 {
					m.selectedSessIdx++
				}
				return m, nil
			case "enter":
				if len(m.sessions) > 0 && m.selectedSessIdx < len(m.sessions) {
					sess := m.sessions[m.selectedSessIdx]
					m.activeSessionID = sess.ID
					m.chatHistory = sess.Messages
					m.viewport.SetContent(m.renderChatTranscript())
					m.activeTab = TabChat
					m.statusMsg = "Switched active thread to " + sess.ID
				}
				return m, nil
			case "d":
				if len(m.sessions) > 0 && m.selectedSessIdx < len(m.sessions) {
					sessID := m.sessions[m.selectedSessIdx].ID
					return m, m.cmdDeleteSession(sessID)
				}
				return m, nil
			}

		case TabAdmin:
			switch msg.String() {
			case "p":
				return m, m.cmdCreateKey("pro")
			case "s":
				return m, m.cmdCreateKey("starter")
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 6
		m.viewport.Height = msg.Height - 12
		m.textarea.SetWidth(msg.Width - 6)

	case TokenMsg:
		m.streamingText.WriteString(string(msg))
		m.viewport.SetContent(m.renderChatTranscript() + "\n" + AssistantMsgStyle.Render("ASSISTANT: ") + m.streamingText.String())
		m.viewport.GotoBottom()
		return m, m.waitForToken()

	case StreamDoneMsg:
		m.isStreaming = false
		fullText := m.streamingText.String()
		m.streamingText.Reset()
		m.chatHistory = append(m.chatHistory, completion.ChatMessage{Role: "assistant", Content: fullText})
		m.viewport.SetContent(m.renderChatTranscript())
		m.viewport.GotoBottom()
		m.statusMsg = "Stream finished."
		return m, m.cmdFetchUsage()

	case StreamErrMsg:
		m.isStreaming = false
		m.statusMsg = "Error: " + msg.err.Error()
		return m, nil

	case UsageLoadedMsg:
		m.tokensUsed = msg.tokens
		return m, nil

	case PersonasLoadedMsg:
		m.personas = msg.personas
		return m, nil

	case SessionsLoadedMsg:
		m.sessions = msg.sessions
		return m, nil

	case KeyCreatedMsg:
		m.apiKey = msg.key
		m.plan = msg.plan
		m.statusMsg = fmt.Sprintf("Created & activated key: %s (%s)", msg.key, msg.plan)
		return m, m.cmdFetchUsage()
	}

	if m.activeTab == TabChat && !m.isStreaming {
		var taCmd tea.Cmd
		m.textarea, taCmd = m.textarea.Update(msg)
		cmds = append(cmds, taCmd)
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var sb strings.Builder

	// Top Banner
	banner := BannerStyle.Render(fmt.Sprintf("⚡ AI SAAS LAB TERMINAL CLIENT  |  Server: %s", m.client.BaseURL))
	sb.WriteString(banner + "\n")

	// Tabs Header
	tabs := []string{" [1] Dashboard ", " [2] Chat ", " [3] Sessions ", " [4] Key Admin "}
	var tabItems []string
	for i, t := range tabs {
		if TabIndex(i) == m.activeTab {
			tabItems = append(tabItems, ActiveTabStyle.Render(t))
		} else {
			tabItems = append(tabItems, InactiveTabStyle.Render(t))
		}
	}
	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabItems...) + "\n\n")

	// Content Area according to activeTab
	switch m.activeTab {
	case TabDashboard:
		sb.WriteString(m.viewDashboard())
	case TabChat:
		sb.WriteString(m.viewChat())
	case TabSessions:
		sb.WriteString(m.viewSessions())
	case TabAdmin:
		sb.WriteString(m.viewAdmin())
	}

	// Status Line & Footer Keybindings
	sb.WriteString("\n" + SystemMsgStyle.Render("Status: "+m.statusMsg))
	sb.WriteString("\n" + HelpFooterStyle.Render("Tab/1-4: Switch Views | Ctrl+C: Quit | Ctrl+R: Persona | Ctrl+M: Memory | Enter: Send"))

	return ContainerBox.Render(sb.String())
}

func (m Model) viewDashboard() string {
	var sb strings.Builder

	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorCyan).Render("📊 SYSTEM DASHBOARD & USAGE OVERVIEW") + "\n\n")

	sb.WriteString(fmt.Sprintf("  • Active API Key : %s\n", lipgloss.NewStyle().Foreground(ColorYellow).Render(m.apiKey)))
	sb.WriteString(fmt.Sprintf("  • Plan Tier      : %s\n", PlanBadge.Render(strings.ToUpper(m.plan))))
	sb.WriteString(fmt.Sprintf("  • Server Status  : %s\n\n", StatusBadge.Render("ONLINE")))

	sb.WriteString("  • Daily Token Quota Gauge:\n")
	sb.WriteString("    " + renderProgressBar(m.tokensUsed, m.dailyQuota, 30) + "\n\n")

	sb.WriteString("  • Active Persona Mode : " + PersonaBadge.Render(m.selectedPersona) + "\n")
	sb.WriteString(fmt.Sprintf("  • Memory Context Recall: %v\n", m.includeMemory))
	sb.WriteString(fmt.Sprintf("  • Active Session ID    : %s\n", m.activeSessionID))

	return sb.String()
}

func (m Model) viewChat() string {
	var sb strings.Builder

	// Top Chat Controls Header
	hdr := fmt.Sprintf("Role Mode: %s  |  Memory Recall: %v  |  Session: %s",
		PersonaBadge.Render(m.selectedPersona),
		m.includeMemory,
		m.activeSessionID,
	)
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorSlate).Render(hdr) + "\n\n")

	// Messages Viewport
	sb.WriteString(m.viewport.View() + "\n\n")

	// Text Input Area
	if m.isStreaming {
		sb.WriteString(SystemMsgStyle.Render("⏳ Streaming tokens from AI completion engine..."))
	} else {
		sb.WriteString(m.textarea.View())
	}

	return sb.String()
}

func (m Model) viewSessions() string {
	var sb strings.Builder

	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorCyan).Render("💬 CHAT SESSIONS MANAGER") + "\n\n")

	if len(m.sessions) == 0 {
		sb.WriteString("  No active conversation sessions found. Start chatting in Tab [2]!\n")
		return sb.String()
	}

	for i, sess := range m.sessions {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(ColorWhite)
		if i == m.selectedSessIdx {
			cursor = "👉"
			style = lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
		}
		if sess.ID == m.activeSessionID {
			cursor += " ★"
		}

		line := fmt.Sprintf("%s %s (%d messages, persona: %s, updated: %s)",
			cursor,
			sess.ID,
			len(sess.Messages),
			sess.Persona,
			sess.UpdatedAt.Format("15:04:05"),
		)
		sb.WriteString(style.Render(line) + "\n")
	}

	sb.WriteString("\n" + HelpFooterStyle.Render("Press [Enter] to switch to selected thread | [d] to delete thread"))
	return sb.String()
}

func (m Model) viewAdmin() string {
	var sb strings.Builder

	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorCyan).Render("🔑 API KEY & PLAN MANAGEMENT") + "\n\n")

	sb.WriteString("  Current Active Key:\n")
	sb.WriteString(fmt.Sprintf("    Key  : %s\n", m.apiKey))
	sb.WriteString(fmt.Sprintf("    Plan : %s\n", m.plan))
	sb.WriteString(fmt.Sprintf("    Usage: %d tokens\n\n", m.tokensUsed))

	sb.WriteString("  Quick Key Issuance Actions:\n")
	sb.WriteString("    • Press [p] to issue a new PRO plan API key\n")
	sb.WriteString("    • Press [s] to issue a new STARTER plan API key\n")

	return sb.String()
}

func (m *Model) cyclePersona() {
	personas := []string{"developer", "analyst", "creative", "tutor", "support"}
	for i, p := range personas {
		if p == m.selectedPersona {
			m.selectedPersona = personas[(i+1)%len(personas)]
			return
		}
	}
	m.selectedPersona = "developer"
}

func (m Model) renderChatTranscript() string {
	var sb strings.Builder
	for _, msg := range m.chatHistory {
		switch msg.Role {
		case "user":
			sb.WriteString(UserMsgStyle.Render("USER: ") + msg.Content + "\n\n")
		case "assistant":
			sb.WriteString(AssistantMsgStyle.Render("ASSISTANT: ") + msg.Content + "\n\n")
		case "system":
			sb.WriteString(SystemMsgStyle.Render("SYSTEM: ") + msg.Content + "\n\n")
		}
	}
	return sb.String()
}

// Commands & Async Effects
func (m *Model) startStreaming(prompt string) tea.Cmd {
	m.isStreaming = true
	m.chatHistory = append(m.chatHistory, completion.ChatMessage{Role: "user", Content: prompt})
	m.viewport.SetContent(m.renderChatTranscript() + "\n" + AssistantMsgStyle.Render("ASSISTANT: ⏳ Thinking..."))
	m.viewport.GotoBottom()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelStream = cancel
	m.tokenChan = make(chan string, 100)
	m.errChan = make(chan error, 10)

	req := completion.ChatRequest{
		APIKey:         m.apiKey,
		Prompt:         prompt,
		SessionID:      m.activeSessionID,
		RoleMode:       m.selectedPersona,
		IncludeHistory: m.includeMemory,
		Stream:         true,
	}

	go m.client.StreamCompletion(ctx, req, m.tokenChan, m.errChan)

	return m.waitForToken()
}

func (m Model) waitForToken() tea.Cmd {
	return func() tea.Msg {
		select {
		case err := <-m.errChan:
			if err != nil {
				return StreamErrMsg{err: err}
			}
		case tok, ok := <-m.tokenChan:
			if !ok {
				return StreamDoneMsg{}
			}
			return TokenMsg(tok)
		}
		return nil
	}
}

func (m Model) cmdFetchUsage() tea.Cmd {
	return func() tea.Msg {
		res, err := m.client.FetchUsage(m.apiKey)
		if err != nil {
			return nil
		}
		return UsageLoadedMsg{tokens: res.Tokens}
	}
}

func (m Model) cmdFetchPersonas() tea.Cmd {
	return func() tea.Msg {
		personas, err := m.client.FetchPersonas()
		if err != nil {
			return nil
		}
		return PersonasLoadedMsg{personas: personas}
	}
}

func (m Model) cmdFetchSessions() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.client.FetchSessions(m.apiKey)
		if err != nil {
			return nil
		}
		return SessionsLoadedMsg{sessions: sessions}
	}
}

func (m Model) cmdCreateKey(plan string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.client.CreateAPIKey(plan)
		if err != nil {
			return nil
		}
		return KeyCreatedMsg{key: res.APIKey, plan: res.Plan}
	}
}

func (m Model) cmdDeleteSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		_ = m.client.DeleteSession(sessionID)
		sessions, err := m.client.FetchSessions(m.apiKey)
		if err != nil {
			return nil
		}
		return SessionsLoadedMsg{sessions: sessions}
	}
}

var ColorYellow = lipgloss.Color("#F1FA8C")
var ColorWhite = lipgloss.Color("#F8F8F2")
