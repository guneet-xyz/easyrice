package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guneet/easyrice/apps/cli/internal/log"
	"github.com/mattn/go-isatty"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // gray
	boldStyle    = lipgloss.NewStyle().Bold(true)

	tickMark  = successStyle.Render("✓")
	crossMark = errorStyle.Render("✗")
)

// StepsMode controls how step details are displayed.
type StepsMode int

const (
	// StepsDefault shows the spinner while running, clears on success, keeps tree on failure.
	StepsDefault StepsMode = iota
	// StepsAlways keeps the step tree visible after completion (success or failure).
	StepsAlways
	// StepsNever hides step details entirely — no spinner, no tree.
	StepsNever
)

var stepsMode StepsMode

// SetStepsMode sets the global steps display mode.
func SetStepsMode(mode StepsMode) {
	stepsMode = mode
}

// Step represents a single step in a multi-step operation.
type Step struct {
	Title string
	Run   func() error
}

// stepDoneMsg is sent when the current step's goroutine finishes.
type stepDoneMsg struct {
	err error
}

// model is the bubbletea model for the step-by-step spinner.
type model struct {
	title   string
	steps   []Step
	current int
	spinner spinner.Model
	done    bool
	err     error
}

func newModel(title string, steps []Step) model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5")) // purple

	return model{
		title:   title,
		steps:   steps,
		current: 0,
		spinner: s,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runCurrentStep())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("interrupted")
			return m, tea.Quit
		}

	case stepDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.done = true
			return m, tea.Quit
		}

		m.current++
		if m.current >= len(m.steps) {
			m.done = true
			return m, tea.Quit
		}
		return m, m.runCurrentStep()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	// When done, return empty — we handle final tree rendering outside bubbletea
	// so that log output (printed above via programWriter) stays on top.
	if m.done {
		return ""
	}

	// In progress: show spinner tree
	return m.renderTree()
}

func (m model) renderTree() string {
	var s string

	// Title header
	if m.done && m.err == nil {
		s += fmt.Sprintf("  %s %s\n", tickMark, boldStyle.Render(m.title))
	} else if m.done && m.err != nil {
		s += fmt.Sprintf("  %s %s\n", crossMark, boldStyle.Render(m.title))
	} else {
		s += fmt.Sprintf("  %s %s\n", m.spinner.View(), boldStyle.Render(m.title))
	}

	// Sub-steps (indented)
	for i, step := range m.steps {
		switch {
		case i < m.current:
			s += fmt.Sprintf("    %s %s\n", tickMark, step.Title)
		case i == m.current:
			if m.err != nil {
				s += fmt.Sprintf("    %s %s\n", crossMark, step.Title)
				s += fmt.Sprintf("      %s\n", errorStyle.Render(m.err.Error()))
			} else {
				s += fmt.Sprintf("    %s %s\n", m.spinner.View(), step.Title)
			}
		default:
			s += fmt.Sprintf("      %s\n", dimStyle.Render(step.Title))
		}
	}

	return s
}

func (m model) runCurrentStep() tea.Cmd {
	step := m.steps[m.current]
	return func() tea.Msg {
		return stepDoneMsg{err: step.Run()}
	}
}

// shouldShowTree returns true if the tree should be rendered after completion.
func shouldShowTree(err error) bool {
	switch stepsMode {
	case StepsNever:
		return false
	case StepsAlways:
		return true
	default:
		// StepsDefault: show on failure only
		return err != nil
	}
}

// programWriter adapts a *tea.Program to io.Writer so charmbracelet/log
// can print above the TUI via Program.Println.
type programWriter struct {
	p *tea.Program
}

func (w *programWriter) Write(b []byte) (int, error) {
	// Println adds a newline, so strip trailing newline from log output
	s := strings.TrimRight(string(b), "\n")
	w.p.Println(s)
	return len(b), nil
}

// runStepsTUI runs steps with the interactive bubbletea spinner.
// Logger output is redirected above the TUI via programWriter so logs
// always appear above the step tree.
func runStepsTUI(title string, steps []Step) error {
	m := newModel(title, steps)
	p := tea.NewProgram(m)

	// Redirect all log output above the TUI during execution
	log.Get().SetOutput(&programWriter{p: p})
	defer log.Get().SetOutput(os.Stderr)

	result, err := p.Run()

	// Restore logger before printing the tree so any further output goes to stderr
	log.Get().SetOutput(os.Stderr)

	if err != nil {
		return fmt.Errorf("tui error: %w", err)
	}

	finalModel, ok := result.(model)
	if !ok {
		return nil
	}

	// Print the final tree below the logs (which were printed above during execution)
	if shouldShowTree(finalModel.err) {
		fmt.Println() // blank line between logs and tree
		fmt.Print(finalModel.renderTree())
	}

	if finalModel.err != nil {
		return finalModel.err
	}

	return nil
}

// runStepsPlain runs steps with plain text output (no TUI).
// Used when stdout is not a terminal (CI, piped output, etc.).
func runStepsPlain(title string, steps []Step) error {
	var failedAt int
	var stepErr error

	for i, step := range steps {
		if err := step.Run(); err != nil {
			failedAt = i
			stepErr = err
			break
		}
	}

	if stepErr != nil && shouldShowTree(stepErr) {
		fmt.Println() // blank line between logs and tree
		fmt.Printf("  ✗ %s\n", title)
		for j, s := range steps {
			switch {
			case j < failedAt:
				fmt.Printf("    ✓ %s\n", s.Title)
			case j == failedAt:
				fmt.Printf("    ✗ %s\n", s.Title)
				fmt.Printf("      %s\n", stepErr.Error())
			default:
				fmt.Printf("      %s\n", s.Title)
			}
		}
		return stepErr
	}

	if stepErr == nil && stepsMode == StepsAlways {
		fmt.Println() // blank line between logs and tree
		fmt.Printf("  ✓ %s\n", title)
		for _, s := range steps {
			fmt.Printf("    ✓ %s\n", s.Title)
		}
	}

	return stepErr
}

// runSilent runs steps without any output.
func runSilent(steps []Step) error {
	for _, step := range steps {
		if err := step.Run(); err != nil {
			return err
		}
	}
	return nil
}

// RunSteps runs the given steps sequentially under a titled group with a spinner.
//
// Default: spinner shows while running, clears on success, stays on failure.
// --steps: spinner shows while running, keeps tree on success and failure.
// --no-steps: no spinner, no tree, steps run silently.
//
// Log output always appears above the step tree.
// A blank line separates logs from the tree when the tree is shown.
//
// Falls back to plain text output if stdout is not a terminal.
func RunSteps(title string, steps []Step) error {
	if stepsMode == StepsNever {
		return runSilent(steps)
	}

	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return runStepsTUI(title, steps)
	}
	return runStepsPlain(title, steps)
}
