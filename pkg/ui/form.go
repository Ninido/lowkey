package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// boundedLayout keeps Huh's automatic resize handling, with a width ceiling.
type boundedLayout struct{ huh.Layout }

func (boundedLayout) GroupWidth(_ *huh.Form, _ *huh.Group, width int) int {
	return max(1, min(60, width-1)) // Leave the final column free to avoid autowrap.
}

// NewForm builds a fixed-width form. Show it with Run.
func NewForm(fields ...huh.Field) *huh.Form {
	return huh.NewForm(huh.NewGroup(fields...).WithWidth(60)).
		WithLayout(boundedLayout{huh.LayoutDefault}).
		WithShowHelp(false)
}

// The whole interactive flow runs inside ONE alt-screen program: the header
// and the current form are laid out from scratch on every frame and resize,
// so the terminal never reflows stale output and forms never move.
var (
	prog    *tea.Program
	exited  chan struct{}
	progErr error
	notes   []string // printed to stdout on Stop so they stay in scrollback
)

const (
	maxNotes     = 3
	minFormRows  = 8 // below this the big splash collapses to a one-line title
	splashMargin = 1
)

type runMsg struct {
	form *huh.Form
	done chan error
}

type noteMsg string

type screen struct {
	splash func(width int) string
	form   *huh.Form
	done   chan error
	notes  []string
	w, h   int
}

func (s *screen) Init() tea.Cmd { return nil }

func (s *screen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		changed := msg.Width != s.w || msg.Height != s.h
		s.w, s.h = msg.Width, msg.Height
		s.layout()
		if changed {
			return s, tea.ClearScreen // wipe whatever the terminal reflowed
		}
		return s, nil
	case runMsg:
		s.form, s.done = msg.form, msg.done
		s.layout()
		return s, s.form.Init()
	case noteMsg:
		s.notes = append(s.notes, string(msg))
		s.layout()
		return s, nil
	case tea.KeyMsg:
		if s.form == nil && msg.Type == tea.KeyCtrlC {
			return s, tea.Quit
		}
	}
	if s.form == nil {
		return s, nil
	}
	_, cmd := s.form.Update(msg)
	switch s.form.State {
	case huh.StateCompleted:
		s.finish(nil)
	case huh.StateAborted:
		s.finish(huh.ErrUserAborted)
	}
	return s, cmd
}

func (s *screen) finish(err error) {
	s.done <- err
	s.form, s.done = nil, nil
}

// layout gives the form exactly the rows left below the header.
func (s *screen) layout() {
	if s.form != nil && s.w > 0 {
		s.form.Update(tea.WindowSizeMsg{Width: s.w, Height: max(1, s.h-lipgloss.Height(s.top())-1)})
	}
}

func (s *screen) top() string {
	head := s.splash(s.w)
	if lipgloss.Height(head)+splashMargin+minFormRows > s.h {
		head = RenderTitle(s.w)
	}
	lines := []string{head}
	for _, n := range s.notes[max(0, len(s.notes)-maxNotes):] {
		lines = append(lines, ansi.Truncate(n, s.w-1, "…"))
	}
	return strings.Join(lines, "\n")
}

func (s *screen) View() string {
	if s.w == 0 {
		return "" // size unknown yet; Bubble Tea sends it right after start
	}
	if s.form == nil {
		return s.top()
	}
	return s.top() + "\n\n" + s.form.View()
}

// Start opens the interactive screen with splash as its header.
func Start(splash func(width int) string) {
	prog = tea.NewProgram(&screen{splash: splash}, tea.WithOutput(os.Stderr), tea.WithAltScreen())
	exited = make(chan struct{})
	go func() {
		_, progErr = prog.Run()
		close(exited)
	}()
}

// Run shows form under the header and blocks until it is submitted or aborted.
func Run(form *huh.Form) error {
	done := make(chan error, 1)
	prog.Send(runMsg{form, done})
	select {
	case err := <-done:
		return err
	case <-exited:
		if progErr != nil {
			return progErr
		}
		return huh.ErrUserAborted
	}
}

// Println shows a status line on screen and prints it to stdout after Stop.
func Println(a ...any) {
	line := strings.TrimSuffix(fmt.Sprintln(a...), "\n")
	if prog == nil {
		fmt.Println(line)
		return
	}
	notes = append(notes, line)
	prog.Send(noteMsg(line))
}

// Stop closes the screen and flushes status lines. Safe to call twice.
func Stop() {
	if prog == nil {
		return
	}
	prog.Quit()
	<-exited
	prog = nil
	for _, n := range notes {
		fmt.Println(n)
	}
	notes = nil
}
