package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pasindu-vishmika/gh-cpr/utils"
)

type state int

const (
	loadingBranches state = iota
	selectSource
	selectTarget
	createNewTarget
	creatingBranch
	enterTitle
	enterBody
	confirmDetails
	creatingPR
	done
	failed
)

// noReturn is the sentinel for "no edit-return step specified".
const noReturn state = -1

// contentWidth is the inner width of the card (excluding border + padding).
const contentWidth = 84

// brandTitle renders "Create Pull Request" with each initial (C, P, R)
// highlighted, so the acronym CPR reads out of the title itself.
func brandTitle() string {
	words := []string{"Create", "Pull", "Request"}
	var b strings.Builder
	for i, w := range words {
		if i > 0 {
			b.WriteString(titleRest.Render(" "))
		}
		b.WriteString(titleHi.Render(w[:1]))
		b.WriteString(titleRest.Render(w[1:]))
	}
	return b.String()
}

// ──────────────────────────────────────────────────────────────────────────────
// # Async messages & commands
// ──────────────────────────────────────────────────────────────────────────────

type branchesLoadedMsg struct {
	items []list.Item
	err   error
}

type branchCreatedMsg struct {
	name string
	err  error
}

type prCreatedMsg struct {
	url string
	err error
}

func fetchBranchesCmd() tea.Msg {
	items, err := getBranches()
	return branchesLoadedMsg{items: items, err: err}
}

func createBranchCmd(name string) tea.Cmd {
	return func() tea.Msg {
		return branchCreatedMsg{name: name, err: utils.CreateBranchFromMain(name)}
	}
}

func createPRCmd(source, target, title, body string) tea.Cmd {
	return func() tea.Msg {
		url, err := utils.ExecutePR(source, target, title, body)
		return prCreatedMsg{url: url, err: err}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// # Styles
// ──────────────────────────────────────────────────────────────────────────────

var (
	accent = lipgloss.Color("#17e7be")
	blue   = lipgloss.Color("#58cff0")
	dim    = lipgloss.Color("#6c6c8a")
	line   = lipgloss.Color("#3b3b4f")
	green  = lipgloss.Color("#32CD32")
	red    = lipgloss.Color("#FF6F6F")

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(line).
			Padding(1, 2)

	titleHi   = lipgloss.NewStyle().Foreground(accent).Bold(true)         // the C / P / R initials
	titleRest = lipgloss.NewStyle().Foreground(lipgloss.Color("#c8c8de")) // the rest of the words
	// Very low-contrast on purpose, and adaptive so it stays a faint whisper on
	// both light and dark terminals.
	// Use ANSI "bright black" (palette color 8): the terminal maps it to a muted
	// grey appropriate for its own theme, so it stays a faint whisper on both
	// light and dark backgrounds without needing background detection (which the
	// VS Code terminal, among others, doesn't reliably answer). Faint dims it
	// further where supported.
	creditStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Faint(true).Italic(true).Width(contentWidth).Align(lipgloss.Center)
	sepStyle   = lipgloss.NewStyle().Foreground(line)
	footerKey  = lipgloss.NewStyle().Foreground(blue).Bold(true)
	footerTxt  = lipgloss.NewStyle().Foreground(dim)
	labelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	hintStyle  = lipgloss.NewStyle().Foreground(dim).Italic(true)

	crumbDone    = lipgloss.NewStyle().Foreground(green)
	crumbActive  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	crumbPending = lipgloss.NewStyle().Foreground(dim)
	crumbSep     = lipgloss.NewStyle().Foreground(line)

	summaryLabel = lipgloss.NewStyle().Foreground(dim).Width(8)
	summaryValue = lipgloss.NewStyle().Foreground(blue)
	numBadge     = lipgloss.NewStyle().Foreground(lipgloss.Color("#0b0f14")).Background(blue).Bold(true).Padding(0, 1)

	okStyle      = lipgloss.NewStyle().Foreground(green).Bold(true)
	urlStyle     = lipgloss.NewStyle().Foreground(blue).Underline(true)
	failStyle    = lipgloss.NewStyle().Foreground(red).Bold(true)
	spinnerStyle = lipgloss.NewStyle().Foreground(accent)
	errorStyle   = lipgloss.NewStyle().Foreground(red).Bold(true)
)

// ──────────────────────────────────────────────────────────────────────────────
// # Branch list plumbing
// ──────────────────────────────────────────────────────────────────────────────

func getBranches() ([]list.Item, error) {
	out, err := exec.Command("git", "ls-remote", "--heads", "origin").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	var items []list.Item

	for _, l := range lines {
		if l == "" {
			continue
		}
		parts := strings.Split(l, "\t")
		if len(parts) < 2 {
			continue
		}
		branch := strings.TrimPrefix(parts[1], "refs/heads/")
		if branch != "" {
			items = append(items, listItem(branch))
		}
	}
	return items, nil
}

type listItem string

func (i listItem) FilterValue() string { return string(i) }
func (i listItem) Title() string       { return string(i) }
func (i listItem) Description() string { return "" }

type customDelegate struct{}

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(3).Foreground(lipgloss.Color("#c8c8de"))
	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(lipgloss.Color("#0b0f14")).
				Background(accent).
				Bold(true)
)

func (d customDelegate) Height() int                               { return 1 }
func (d customDelegate) Spacing() int                              { return 0 }
func (d customDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d customDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	li, ok := item.(listItem)
	if !ok {
		return
	}
	if index == m.Index() {
		// Render the selection as a full-width highlighted bar so it stands out.
		fmt.Fprint(w, selectedItemStyle.Width(m.Width()).Render("❯ "+li.Title()))
	} else {
		fmt.Fprint(w, itemStyle.Render(li.Title()))
	}
}

func newBranchList(title string) list.Model {
	l := list.New(nil, customDelegate{}, contentWidth, 11)
	l.Title = title
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Foreground(accent).Bold(true)
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(dim)
	l.KeyMap.Quit = key.NewBinding()      // disable default 'q'
	l.KeyMap.ForceQuit = key.NewBinding() // disable ctrl+c quitting from the list
	l.KeyMap.Filter = key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	)
	return l
}

// ──────────────────────────────────────────────────────────────────────────────
// # Model
// ──────────────────────────────────────────────────────────────────────────────

type model struct {
	step       state
	returnStep state

	sourceList     list.Model
	targetList     list.Model
	titleInput     textinput.Model
	bodyInput      textarea.Model
	newBranchInput textinput.Model
	spinner        spinner.Model

	currentBranch string
	prTitle       string
	prBody        string
	loadingLabel  string

	prURL  string
	err    error
	errMsg string // inline validation message

	width  int
	height int
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchBranchesCmd)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		// shift+tab steps back to the previous step from anywhere (no-op on the
		// first step and on non-navigable screens). Intercepted here so text
		// inputs never see it.
		if msg.String() == "shift+tab" {
			if nm, atFirst := m.back(); !atFirst {
				return nm, nil
			}
			return m, nil
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case branchesLoadedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("unable to retrieve branches from the remote repository:\n%w", msg.err)
			m.step = failed
			return m, nil
		}
		if len(msg.items) == 0 {
			m.err = fmt.Errorf("no branches found in the remote repository")
			m.step = failed
			return m, nil
		}
		m.sourceList.SetItems(msg.items)
		m.targetList.SetItems(msg.items)
		selectBranch(&m.sourceList, m.currentBranch)
		selectBranch(&m.targetList, "main")
		m.step = selectSource
		return m, nil

	case branchCreatedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("could not create branch %q:\n%w", msg.name, msg.err)
			m.step = failed
			return m, nil
		}
		newItem := listItem(msg.name)
		m.sourceList.SetItems(append(m.sourceList.Items(), newItem))
		m.targetList.SetItems(append(m.targetList.Items(), newItem))
		selectBranch(&m.targetList, msg.name)
		m.step = selectTarget
		return m, nil

	case prCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.step = failed
			return m, nil
		}
		m.prURL = msg.url
		m.step = done
		return m, nil
	}

	return m.updateStep(msg)
}

func (m model) updateStep(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.step {

	case selectSource:
		// Capture the filter state *before* the list processes the key, so we
		// can tell whether esc/enter belong to the filter or to the wizard.
		fs := m.sourceList.FilterState()
		m.sourceList, cmd = m.sourceList.Update(msg)
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "esc":
				// esc quits, but only when no filter is active; otherwise the
				// list just cancelled/cleared the filter and we stay put.
				if fs == list.Unfiltered {
					return m, tea.Quit
				}
			case "enter":
				// While typing a filter, enter applies it (handled by the list).
				if fs != list.Filtering {
					if m.returnStep == confirmDetails {
						m.step, m.returnStep = confirmDetails, noReturn
					} else {
						m.step = selectTarget
					}
				}
			}
		}
		return m, cmd

	case selectTarget:
		fs := m.targetList.FilterState()
		m.targetList, cmd = m.targetList.Update(msg)
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "esc":
				if fs == list.Unfiltered {
					return m, tea.Quit
				}
			case "ctrl+n":
				if fs == list.Filtering {
					break // let the filter input keep the keystroke
				}
				m.newBranchInput.SetValue("")
				m.newBranchInput.Focus()
				m.step = createNewTarget
			case "enter":
				if fs == list.Filtering {
					break // applying the filter, not selecting
				}
				if m.returnStep == confirmDetails {
					m.step, m.returnStep = confirmDetails, noReturn
				} else {
					// Only auto-fill a default title when the field is still
					// empty, so stepping back and forward never clobbers a
					// title the user has already typed.
					if strings.TrimSpace(m.titleInput.Value()) == "" {
						src := m.sourceList.SelectedItem().(listItem)
						tgt := m.targetList.SelectedItem().(listItem)
						if def, err := utils.GetDefaultPRTitle(string(src), string(tgt)); err == nil && strings.TrimSpace(def) != "" {
							m.titleInput.SetValue(def)
						}
					}
					m.titleInput.CursorEnd()
					m.titleInput.Focus()
					m.step = enterTitle
				}
			}
		}
		return m, cmd

	case createNewTarget:
		m.newBranchInput, cmd = m.newBranchInput.Update(msg)
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "esc":
				m.step = selectTarget
			case "enter":
				name := strings.TrimSpace(m.newBranchInput.Value())
				if name == "" {
					return m, nil
				}
				m.loadingLabel = "Creating branch " + name + " from origin/main"
				m.step = creatingBranch
				return m, createBranchCmd(name)
			}
		}
		return m, cmd

	case enterTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "esc":
				return m, tea.Quit
			case "enter":
				title := strings.TrimSpace(m.titleInput.Value())
				if title == "" {
					m.errMsg = "A pull request title is required."
					return m, nil
				}
				m.prTitle, m.errMsg = title, ""
				if m.returnStep == confirmDetails {
					m.step, m.returnStep = confirmDetails, noReturn
				} else {
					m.bodyInput.Focus()
					m.step = enterBody
				}
			}
		}
		return m, cmd

	case enterBody:
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "esc":
				return m, tea.Quit
			case "enter":
				// Enter submits; newlines come from Ctrl+J / Ctrl+Enter.
				m.prBody = m.bodyInput.Value()
				m.step, m.returnStep = confirmDetails, noReturn
				return m, nil
			}
		}
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		return m, cmd

	case confirmDetails:
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "esc":
				return m, tea.Quit
			case "1":
				m.returnStep, m.step = confirmDetails, selectSource
			case "2":
				m.returnStep, m.step = confirmDetails, selectTarget
			case "3":
				m.returnStep, m.step = confirmDetails, enterTitle
				m.titleInput.CursorEnd()
				m.titleInput.Focus()
			case "4":
				m.returnStep, m.step = confirmDetails, enterBody
				m.bodyInput.Focus()
			case "c":
				src := m.sourceList.SelectedItem().(listItem)
				tgt := m.targetList.SelectedItem().(listItem)
				m.loadingLabel = "Creating pull request"
				m.step = creatingPR
				return m, createPRCmd(string(src), string(tgt), m.prTitle, m.prBody)
			}
		}
		return m, nil

	case done, failed:
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "enter", "esc", "q":
				return m, tea.Quit
			}
		}
		return m, nil
	}

	return m, nil
}

func selectBranch(l *list.Model, name string) {
	for i, itm := range l.Items() {
		if li, ok := itm.(listItem); ok && li.Title() == name {
			l.Select(i)
			return
		}
	}
}

// back moves to the previous wizard step. It returns atFirst == true when there
// is no previous step, so the caller can treat it as a no-op.
func (m model) back() (model, bool) {
	// If we jumped in to edit a single field from the confirm screen, going
	// back just cancels that edit and returns to the summary.
	if m.returnStep == confirmDetails {
		m.step, m.returnStep = confirmDetails, noReturn
		return m, false
	}

	switch m.step {
	case selectTarget:
		m.step = selectSource
	case enterTitle:
		m.titleInput.Blur()
		m.step = selectTarget
	case enterBody:
		m.bodyInput.Blur()
		m.titleInput.Focus()
		m.step = enterTitle
	case confirmDetails:
		m.bodyInput.Focus()
		m.step = enterBody
	default:
		return m, true // selectSource / no previous step → quit
	}
	return m, false
}

// ──────────────────────────────────────────────────────────────────────────────
// # View
// ──────────────────────────────────────────────────────────────────────────────

func (m model) View() string {
	body, footer := m.content()
	return m.chrome(body, footer)
}

// content returns the body and footer strings for the current step.
func (m model) content() (string, string) {
	var body, footer string

	switch m.step {
	case loadingBranches:
		body = m.spinner.View() + spinnerStyle.Render(" Fetching branches from origin…")
		footer = keyHint("ctrl+c", "quit")

	case selectSource:
		body = m.sourceList.View()
		switch m.sourceList.FilterState() {
		case list.Filtering:
			footer = keys("↵", "apply", "esc", "cancel search")
		case list.FilterApplied:
			footer = keys("↑/↓", "move", "↵", "select", "esc", "clear filter")
		default:
			footer = keys("↑/↓", "move", "↵", "select", "/", "search", "esc", "quit")
		}

	case selectTarget:
		body = m.targetList.View()
		switch m.targetList.FilterState() {
		case list.Filtering:
			footer = keys("↵", "apply", "esc", "cancel search")
		case list.FilterApplied:
			footer = keys("↑/↓", "move", "↵", "select", "ctrl+n", "new branch", "esc", "clear filter")
		default:
			footer = keys("↑/↓", "move", "↵", "select", "ctrl+n", "new branch", "shift+tab", "back", "esc", "quit")
		}

	case createNewTarget:
		body = labelStyle.Render("New branch name") + hintStyle.Render("  (branched from origin/main)") +
			"\n\n" + m.newBranchInput.View()
		footer = keys("↵", "create", "esc", "cancel")

	case creatingBranch:
		body = m.spinner.View() + spinnerStyle.Render(" "+m.loadingLabel+"…")
		footer = footerTxt.Render("please wait")

	case enterTitle:
		body = labelStyle.Render("Pull request title") + hintStyle.Render("  (required)") +
			"\n\n" + m.titleInput.View()
		if m.errMsg != "" {
			body += "\n\n" + errorStyle.Render(m.errMsg)
		}
		footer = keys("↵", "confirm", "shift+tab", "back", "esc", "quit")

	case enterBody:
		body = labelStyle.Render("Pull request body") + hintStyle.Render("  (optional)") +
			"\n\n" + m.bodyInput.View()
		footer = keys("↵", "done", "ctrl+j / ctrl+↵", "new line", "shift+tab", "back", "esc", "quit")

	case confirmDetails:
		src := m.sourceList.SelectedItem().(listItem)
		tgt := m.targetList.SelectedItem().(listItem)
		body = labelStyle.Render("Review your pull request") + "\n\n" +
			summaryRow(1, "Source", string(src)) +
			summaryRow(2, "Target", string(tgt)) +
			summaryRow(3, "Title", m.prTitle) +
			summaryRow(4, "Body", bodyPreview(m.prBody)) + "\n" +
			hintStyle.Render("Press a number (1–4) to edit that field, or:")
		footer = keys("c", "create pull request", "1–4", "edit field", "shift+tab", "back", "esc", "quit")

	case creatingPR:
		body = m.spinner.View() + spinnerStyle.Render(" "+m.loadingLabel+"…")
		footer = footerTxt.Render("please wait")

	case done:
		body = okStyle.Render("✓ Pull request created") + "\n\n"
		if m.prURL != "" {
			body += urlStyle.Render(m.prURL)
		} else {
			body += footerTxt.Render("(the PR was created; no URL was returned)")
		}
		footer = keys("↵/esc", "close")

	case failed:
		body = failStyle.Render("✗ Could not create the pull request") + "\n\n" +
			lipgloss.NewStyle().Foreground(dim).Width(contentWidth).Render(m.err.Error())
		footer = keys("↵/esc", "close")
	}

	return body, footer
}

// card assembles the bordered card (without positioning it in the terminal).
func (m model) card(body, footer string) string {
	header := lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, brandTitle())
	sep := sepStyle.Render(strings.Repeat("─", contentWidth))
	credit := creditStyle.Render("Developed by Pasindu-Vishmika")

	parts := []string{header}
	if bc := m.breadcrumb(); bc != "" {
		parts = append(parts, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, bc))
	}
	parts = append(parts, sep, "", body, "", footer, sep, credit)

	return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m model) chrome(body, footer string) string {
	card := m.card(body, footer)
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
	}
	return card
}

func (m model) breadcrumb() string {
	if m.step == loadingBranches || m.step == failed {
		return ""
	}
	steps := []string{"Source", "Target", "Title", "Body", "Confirm"}
	cur := stepIndex(m.step)

	parts := make([]string, len(steps))
	for i, s := range steps {
		switch {
		case i < cur:
			parts[i] = crumbDone.Render("● " + s)
		case i == cur:
			parts[i] = crumbActive.Render("● " + s)
		default:
			parts[i] = crumbPending.Render("○ " + s)
		}
	}
	return strings.Join(parts, crumbSep.Render(" ── "))
}

func stepIndex(s state) int {
	switch s {
	case selectSource:
		return 0
	case selectTarget, createNewTarget, creatingBranch:
		return 1
	case enterTitle:
		return 2
	case enterBody:
		return 3
	case confirmDetails, creatingPR:
		return 4
	case done:
		return 5
	default:
		return -1
	}
}

// valueColumn is the horizontal offset (in cells) where a summary row's value
// begins: the number badge, a space, and the fixed-width label.
func valueColumn() int {
	return lipgloss.Width(numBadge.Render("0")) + 1 + summaryLabel.GetWidth()
}

func summaryRow(num int, label, value string) string {
	if value == "" {
		value = "—"
	}
	prefix := numBadge.Render(strconv.Itoa(num)) + " " + summaryLabel.Render(label)
	return prefix + summaryValue.Render(value) + "\n"
}

func bodyPreview(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	const maxLines = 3
	// Continuation lines align under the value column (past the badge + label).
	indent := strings.Repeat(" ", valueColumn())

	lines := strings.Split(body, "\n")
	shown := lines
	if len(lines) > maxLines {
		shown = lines[:maxLines]
	}

	preview := strings.Join(shown, "\n"+indent)
	if len(lines) > maxLines {
		preview += "\n" + indent + hintStyle.Render(fmt.Sprintf("(+%d more lines)", len(lines)-maxLines))
	}
	return preview
}

// keyHint renders a single "key action" hint.
func keyHint(k, action string) string {
	return footerKey.Render(k) + footerTxt.Render(" "+action)
}

// keys renders alternating key/action pairs joined by dots.
func keys(pairs ...string) string {
	var out []string
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, keyHint(pairs[i], pairs[i+1]))
	}
	return strings.Join(out, footerTxt.Render("  ·  "))
}

// ──────────────────────────────────────────────────────────────────────────────
// # main
// ──────────────────────────────────────────────────────────────────────────────

func main() {
	if !utils.IsGitRepository() {
		fmt.Println(errorStyle.Render(
			"No Git repository found.\n" +
				"CPR (Create-Pull-Request) does not work outside a Git repository.\n" +
				"Navigate to a Git repository and try again."))
		os.Exit(1)
	}

	current, err := utils.GetCurrentBranch()
	if err != nil {
		fmt.Println(errorStyle.Render(err.Error()))
		os.Exit(1)
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	titleInput := textinput.New()
	titleInput.Placeholder = "Add a concise title"
	titleInput.Width = contentWidth - 4
	titleInput.Prompt = "❯ "
	titleInput.PromptStyle = lipgloss.NewStyle().Foreground(accent)

	bodyInput := textarea.New()
	bodyInput.Placeholder = "Describe what changed and why…"
	bodyInput.SetWidth(contentWidth)
	bodyInput.SetHeight(8)
	// Enter submits the body (handled in Update); a new line is inserted with
	// Ctrl+J (universally supported) or Ctrl+Enter (on terminals that report it).
	bodyInput.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("ctrl+j", "ctrl+enter"),
		key.WithHelp("ctrl+j", "new line"),
	)

	newBranchInput := textinput.New()
	newBranchInput.Placeholder = "new-branch-name"
	newBranchInput.Width = contentWidth - 4
	newBranchInput.Prompt = "❯ "
	newBranchInput.PromptStyle = lipgloss.NewStyle().Foreground(accent)

	m := model{
		step:           loadingBranches,
		returnStep:     noReturn,
		sourceList:     newBranchList("Select the source branch"),
		targetList:     newBranchList("Select the target branch"),
		titleInput:     titleInput,
		bodyInput:      bodyInput,
		newBranchInput: newBranchInput,
		spinner:        sp,
		currentBranch:  current,
	}

	fm, runErr := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if runErr != nil {
		fmt.Println(errorStyle.Render("Error running program: " + runErr.Error()))
		os.Exit(1)
	}

	// After the alt screen is torn down, persist the outcome in the scrollback.
	if final, ok := fm.(model); ok {
		switch {
		case final.err != nil:
			fmt.Fprintln(os.Stderr, errorStyle.Render(final.err.Error()))
			os.Exit(1)
		case final.prURL != "":
			fmt.Println(okStyle.Render("✓ Pull request created"))
			fmt.Println(final.prURL)
		}
	}
}
