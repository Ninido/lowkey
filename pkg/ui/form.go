package ui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// boundedLayout keeps Huh's automatic resize handling, with a width ceiling.
type boundedLayout struct{ huh.Layout }

func (boundedLayout) GroupWidth(_ *huh.Form, _ *huh.Group, width int) int {
	return max(1, min(60, width-1)) // Leave the final column free to avoid autowrap.
}

// NewForm uses an alternate screen so terminal reflow cannot displace old rows.
func NewForm(fields ...huh.Field) *huh.Form {
	return huh.NewForm(huh.NewGroup(fields...).WithWidth(60)).
		WithLayout(boundedLayout{huh.LayoutDefault}).
		WithShowHelp(false).
		WithProgramOptions(tea.WithOutput(os.Stderr), tea.WithReportFocus(), tea.WithAltScreen())
}
