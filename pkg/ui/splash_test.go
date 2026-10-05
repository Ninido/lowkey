package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderSplash(t *testing.T) {
	out := RenderSplash(4, 2, false)
	if !strings.Contains(out, "THERMAL ORCHESTRATOR") {
		t.Errorf("expected splash to contain 'THERMAL ORCHESTRATOR', got: %s", out)
	}
	if !strings.Contains(out, "4 Engines Ready") {
		t.Errorf("expected splash to contain '4 Engines Ready', got: %s", out)
	}
	if !strings.Contains(out, "2 Saved Profiles") {
		t.Errorf("expected splash to contain '2 Saved Profiles', got: %s", out)
	}
	if !strings.Contains(out, "AC Connected") {
		t.Errorf("expected splash to contain 'AC Connected', got: %s", out)
	}

	// Test battery state
	batOut := RenderSplash(0, 0, true)
	if !strings.Contains(batOut, "Battery Active") {
		t.Errorf("expected splash to contain 'Battery Active', got: %s", batOut)
	}
	if !strings.Contains(batOut, "0 Engines Detected") {
		t.Errorf("expected splash to contain '0 Engines Detected', got: %s", batOut)
	}
}

func TestSplashFitsTerminal(t *testing.T) {
	for _, width := range []int{30, 59, 60, 80, 120, 160, 200} {
		out := renderSplash(width, 4, 5, true)
		if got := lipgloss.Width(out); got >= width {
			t.Errorf("splash width %d exceeds safe width for %d-column terminal", got, width)
		}
		if !strings.Contains(out, "LOWKEY") && width < 60 && !strings.Contains(out, "|_____") {
			t.Errorf("compact logo missing at width %d", width)
		}
	}
}

func TestRenderGradientLine(t *testing.T) {
	rendered := renderGradientLine("LOWKEY")
	if len(rendered) == 0 {
		t.Errorf("expected rendered gradient to not be empty")
	}
}
