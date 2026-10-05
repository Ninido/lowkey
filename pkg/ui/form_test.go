package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func TestFormResizesWithoutLosingInput(t *testing.T) {
	var name string
	input := huh.NewInput().Title("Setup Profile Name:").Value(&name)
	form := NewForm(input)
	form.Init()
	input.Focus()
	for _, width := range []int{160, 40, 200, 30, 80, 25, 160} {
		form.Update(tea.WindowSizeMsg{Width: width, Height: 40})
		if got, want := lipgloss.Width(form.View()), min(60, width-1); got != want {
			t.Fatalf("resize to %d: form width %d, expected %d", width, got, want)
		}
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	}
	if name != "xxxxxxx" {
		t.Fatalf("resizing lost input: %q", name)
	}
}

func TestProfileInputKeepsFormWidth(t *testing.T) {
	var name string
	input := huh.NewInput().Title("Setup Profile Name:").
		Placeholder("e.g. daily-coding, quiet-agent, heavy-throughput").Value(&name)
	form := NewForm(input)
	form.Init()
	input.Focus()
	for _, terminalWidth := range []int{160, 200, 80} {
		form.Update(tea.WindowSizeMsg{Width: terminalWidth, Height: 40})
		if got := lipgloss.Width(form.View()); got != 60 {
			t.Fatalf("terminal width %d overrode form width: %d", terminalWidth, got)
		}
	}
	height := lipgloss.Height(form.View())
	for _, r := range "erqwerqwe" + strings.Repeat("x", 80) {
		form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		if got := lipgloss.Width(form.View()); got != 60 {
			t.Fatalf("input width changed while typing: %d", got)
		}
		if got := lipgloss.Height(form.View()); got != height {
			t.Fatalf("input wrapped while typing: height %d, expected %d", got, height)
		}
	}
	if !strings.HasPrefix(name, "erqwerqwe") {
		t.Fatalf("typing did not reach the input: %q", name)
	}
}
