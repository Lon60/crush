package dialog

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/pkg/catwalk"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/ui/common"
	uv "github.com/charmbracelet/ultraviolet"
)

type claudeImportState int

const (
	claudeImportSuccess claudeImportState = iota
	claudeImportNotFound
)

const ClaudeOAuthID = "claude-oauth"

// ClaudeOAuth handles Claude Code credential import from disk.
type ClaudeOAuth struct {
	com          *common.Common
	isOnboarding bool

	provider  catwalk.Provider
	model     config.SelectedModel
	modelType config.SelectedModelType

	state claudeImportState
	help  help.Model
	width int

	keyMap struct {
		Submit key.Binding
		Close  key.Binding
	}
}

var _ Dialog = (*ClaudeOAuth)(nil)

// NewOAuthClaude creates a new Claude credential import dialog.
// It attempts to import credentials from the Claude Code CLI.
// If found, it auto-selects the model. If not, it shows instructions.
func NewOAuthClaude(
	com *common.Common,
	isOnboarding bool,
	provider catwalk.Provider,
	model config.SelectedModel,
	modelType config.SelectedModelType,
) (*ClaudeOAuth, tea.Cmd) {
	t := com.Styles

	hlp := help.New()
	hlp.Styles = t.DialogHelpStyles()

	m := &ClaudeOAuth{
		com:          com,
		isOnboarding: isOnboarding,
		provider:     provider,
		model:        model,
		modelType:    modelType,
		state:        claudeImportNotFound,
		help:         hlp,
		width:        60,
	}

	m.keyMap.Submit = key.NewBinding(
		key.WithKeys("enter", "ctrl+y"),
		key.WithHelp("enter", "continue"),
	)
	m.keyMap.Close = CloseKey

	token, ok := com.Store().ImportClaudeCode()
	if ok && token != nil {
		m.state = claudeImportSuccess
	}

	return m, nil
}

func (m *ClaudeOAuth) ID() string {
	return ClaudeOAuthID
}

func (m *ClaudeOAuth) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keyMap.Close):
			return ActionClose{}
		case key.Matches(msg, m.keyMap.Submit):
			if m.state == claudeImportSuccess {
				return ActionSelectModel{
					Provider:  m.provider,
					Model:     m.model,
					ModelType: m.modelType,
				}
			}
			return ActionClose{}
		}
	}
	return nil
}

func (m *ClaudeOAuth) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	t := m.com.Styles
	dialogStyle := t.Dialog.View.Width(m.width)

	content := m.dialogContent()

	if m.isOnboarding {
		DrawOnboarding(scr, area, content)
	} else {
		view := dialogStyle.Render(content)
		DrawCenter(scr, area, view)
	}
	return nil
}

func (m *ClaudeOAuth) dialogContent() string {
	t := m.com.Styles
	helpStyle := t.Dialog.HelpView

	elements := []string{
		m.headerContent(),
		m.innerContent(),
		helpStyle.Render(m.help.View(m)),
	}
	return strings.Join(elements, "\n")
}

func (m *ClaudeOAuth) headerContent() string {
	t := m.com.Styles
	titleStyle := t.Dialog.Title
	textStyle := t.Dialog.PrimaryText
	dialogStyle := t.Dialog.View.Width(m.width)
	headerOffset := titleStyle.GetHorizontalFrameSize() + dialogStyle.GetHorizontalFrameSize()
	dialogTitle := "Claude Code"

	if m.isOnboarding {
		return textStyle.Render(dialogTitle)
	}
	return common.DialogTitle(t, titleStyle.Render(dialogTitle), m.width-headerOffset, t.Primary, t.Secondary)
}

func (m *ClaudeOAuth) innerContent() string {
	t := m.com.Styles
	successStyle := lipgloss.NewStyle().Foreground(t.GreenLight)
	mutedStyle := lipgloss.NewStyle().Foreground(t.FgMuted)
	whiteStyle := lipgloss.NewStyle().Foreground(t.White)

	st := lipgloss.NewStyle().Margin(0, 1).Width(m.width - 2)

	switch m.state {
	case claudeImportSuccess:
		return lipgloss.JoinVertical(lipgloss.Left,
			"",
			st.Render(successStyle.Render("Credentials imported from Claude Code.")),
			"",
			st.Render(mutedStyle.Render("Press enter to continue.")),
			"",
		)

	case claudeImportNotFound:
		return lipgloss.JoinVertical(lipgloss.Left,
			"",
			st.Render(whiteStyle.Render("Claude Code credentials not found.")),
			"",
			st.Render(mutedStyle.Render(fmt.Sprintf(
				"Install Claude Code and run %s first.",
				lipgloss.NewStyle().Foreground(t.Primary).Render("claude auth login"),
			))),
			"",
			st.Render(mutedStyle.Render("Or use the API key option instead.")),
			"",
		)

	default:
		return ""
	}
}

func (m *ClaudeOAuth) ShortHelp() []key.Binding {
	if m.state == claudeImportSuccess {
		return []key.Binding{m.keyMap.Submit}
	}
	return []key.Binding{m.keyMap.Close}
}

func (m *ClaudeOAuth) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}
