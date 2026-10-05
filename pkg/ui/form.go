package ui

import (
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
)

// NewForm sizes the form itself; Field.Run would overwrite a field's width.
func NewForm(fields ...huh.Field) *huh.Form {
	width := 60
	if w, _, err := term.GetSize(os.Stderr.Fd()); err == nil && w > 1 {
		width = min(width, w-1) // Leave the last terminal column free to avoid autowrap.
	}
	return huh.NewForm(huh.NewGroup(fields...)).WithWidth(width).WithShowHelp(false)
}
