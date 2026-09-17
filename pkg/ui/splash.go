package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

var (
	// Gradient colors for LOWKEY banner (Cyber-electric violet -> cyan -> neon teal)
	c1 = lipgloss.Color("#7D56F4") // Electric Violet
	c2 = lipgloss.Color("#8A67F6")
	c3 = lipgloss.Color("#9D78F7")
	c4 = lipgloss.Color("#7088F5")
	c5 = lipgloss.Color("#44A6F3")
	c6 = lipgloss.Color("#00C9E8") // Neon Cyan
	c7 = lipgloss.Color("#04D8A5") // Neon Emerald / Mint

	bannerRows = []string{
		"██╗      ██████╗ ██╗    ██╗██╗  ██╗███████╗██╗   ██╗",
		"██║     ██╔═══██╗██║    ██║██║ ██╔╝██╔════╝╚██╗ ██╔╝",
		"██║     ██║   ██║██║ █╗ ██║█████═╝ █████╗   ╚████╔╝ ",
		"██║     ██║   ██║██║███╗██║██╔═██╗ ██╔══╝    ╚██╔╝  ",
		"███████╗╚██████╔╝╚███╔███╔╝██║ ╚██╗███████╗   ██║   ",
		"╚══════╝ ╚═════╝  ╚══╝╚══╝ ╚═╝  ╚═╝╚══════╝   ╚═╝   ",
	}

	gradColors = []lipgloss.Color{c1, c2, c3, c4, c5, c6, c7}

	// Compact stylized banner for smaller screens
	compactBanner = ` _     _____ _ _ _ _____ _____ __ __
| |   |     | | | |  |  |   __|  |  |
| |___|  |  | | | |    <|   __|_   _|
|_____|_____|_____|__|__|_____| |_|  `
)

// RenderSplash returns the stylized ASCII art, tagline, badges, and quick stats box.
func RenderSplash(detectedEnginesCount int, savedProfilesCount int, onBattery bool) string {
	width := 80
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 20 {
		width = w
	}

	// 1. Render gradient ASCII logo
	var logoLines []string
	if width < 60 {
		style := lipgloss.NewStyle().Foreground(c1).Bold(true)
		logoLines = append(logoLines, style.Render(compactBanner))
	} else {
		// Multi-color horizontal gradient across each line of the block ASCII
		for _, row := range bannerRows {
			logoLines = append(logoLines, renderGradientLine(row))
		}
	}
	logoBlock := strings.Join(logoLines, "\n")

	// 2. Tagline with glowing accent
	taglineAccent := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00F0FF"))
	taglineMuted := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888899"))

	tagline := fmt.Sprintf("%s  %s",
		taglineAccent.Render("⚡ THERMAL ORCHESTRATOR & LOCAL LLM LAUNCHER"),
		taglineMuted.Render("•  Silent · Cool · Battery-Friendly"),
	)

	// 3. Status pills / badges
	badgeStyle := func(bg, fg string, text string) string {
		return lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color(bg)).
			Foreground(lipgloss.Color(fg)).
			Padding(0, 1).
			Render(text)
	}

	var engineBadge string
	if detectedEnginesCount > 0 {
		engineBadge = badgeStyle("#1E3A2F", "#04D8A5", fmt.Sprintf("⚡ %d Engines Ready", detectedEnginesCount))
	} else {
		engineBadge = badgeStyle("#3A2A1E", "#F5A623", "⚠ 0 Engines Detected")
	}

	var profileBadge string
	if savedProfilesCount > 0 {
		profileBadge = badgeStyle("#2B224F", "#C4A7E7", fmt.Sprintf("📂 %d Saved Profiles", savedProfilesCount))
	} else {
		profileBadge = badgeStyle("#22222E", "#888899", "📂 0 Profiles")
	}

	var powerBadge string
	if onBattery {
		powerBadge = badgeStyle("#4A2328", "#FF6B8B", "🔋 Battery Active")
	} else {
		powerBadge = badgeStyle("#183040", "#38BDF8", "🔌 AC Connected")
	}

	qosBadge := badgeStyle("#1A2333", "#818CF8", "🛡 Background QoS")

	badgesRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		engineBadge, "  ",
		profileBadge, "  ",
		powerBadge, "  ",
		qosBadge,
	)

	// 4. Feature Highlights Line
	featLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#555566"))
	featDot := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(" • ")
	featVal := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))

	features := strings.Join([]string{
		featLabel.Render("Duty-Cycle:") + " " + featVal.Render("Micro-pause throttling"),
		featLabel.Render("Thermal:") + " " + featVal.Render("< 60°C silent fans"),
		featLabel.Render("Targets:") + " " + featVal.Render("mtplx · llama.cpp · ollama · mlx · vLLM · lms"),
	}, featDot)

	// 5. Container card with rounded borders and modern neon edge
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		logoBlock,
		"",
		tagline,
		"",
		badgesRow,
		"",
		features,
	)

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		MarginBottom(1)

	return cardStyle.Render(content)
}

// renderGradientLine renders a string with an interpolating color gradient across its characters
func renderGradientLine(line string) string {
	runes := []rune(line)
	n := len(runes)
	if n == 0 {
		return ""
	}

	var out strings.Builder
	numColors := len(gradColors)

	for i, r := range runes {
		colorIdx := (i * numColors) / n
		if colorIdx >= numColors {
			colorIdx = numColors - 1
		}
		c := gradColors[colorIdx]
		style := lipgloss.NewStyle().Foreground(c).Bold(true)
		out.WriteString(style.Render(string(r)))
	}

	return out.String()
}
