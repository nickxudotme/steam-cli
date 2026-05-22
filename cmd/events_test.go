package cmd

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"

	"github.com/charmbracelet/lipgloss"
)

func TestWrapEventDescriptionDoesNotTruncate(t *testing.T) {
	input := "Steam 新品节是一个为期多日的庆祝活动，每年举行三次。在此期间，粉丝们可以体验试用版、与开发者畅聊、观看实况直播，并了解 Steam 上即将推出的游戏。"
	got := wrapEventDescription(input, 40)
	if strings.Contains(got, "...") {
		t.Fatalf("wrapEventDescription truncated with ellipsis: %q", got)
	}
	if removeWhitespace(got) != removeWhitespace(input) {
		t.Fatalf("wrapEventDescription lost content:\n got %q\nwant %q", got, input)
	}
	for _, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 40 {
			t.Fatalf("line width = %d, want <= 40: %q", width, line)
		}
	}
}

func removeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), "")
}

func TestEventDescriptionWidthAdaptsToTerminal(t *testing.T) {
	events := []steam.Event{
		{
			StartDate: "2026-06-15",
			EndDate:   "2026-06-22",
			Status:    "upcoming",
			Category:  "next_fest",
			Name:      "新品节",
		},
	}

	narrow := eventDescriptionWidth(events, 100)
	wide := eventDescriptionWidth(events, 180)
	if narrow >= wide {
		t.Fatalf("description width did not grow with terminal: narrow=%d wide=%d", narrow, wide)
	}
	if narrow < minEventDescriptionWidth {
		t.Fatalf("narrow width = %d, want at least %d", narrow, minEventDescriptionWidth)
	}
	veryWide := eventDescriptionWidth(events, 240)
	if veryWide <= wide {
		t.Fatalf("description width did not keep growing: wide=%d veryWide=%d", wide, veryWide)
	}
}

func TestTerminalWidthFallsBackToColumns(t *testing.T) {
	got := terminalWidthFrom(
		func() (int, error) { return 0, errors.New("not a terminal") },
		func(key string) string {
			if key == "COLUMNS" {
				return "177"
			}
			return ""
		},
	)
	if got != 177 {
		t.Fatalf("terminalWidth() = %d, want COLUMNS fallback", got)
	}
}

func TestRenderEventsUsesLocalizedOfficialPageLabel(t *testing.T) {
	i18n.Set("zh-CN")
	defer i18n.Set("en")

	output := captureStdout(t, func() {
		if err := renderEvents([]steam.Event{
			{
				Name:        "海洋游戏节",
				StartDate:   "2026-05-18",
				EndDate:     "2026-05-25",
				Status:      "active",
				Category:    "fest",
				Description: "关于大海的游戏。",
			},
		}); err != nil {
			t.Fatalf("renderEvents returned error: %v", err)
		}
	})

	if !strings.Contains(output, "Steamworks 官方页面") {
		t.Fatalf("localized official page label missing from output:\n%s", output)
	}
	if strings.Contains(output, "Official Steamworks page") {
		t.Fatalf("English official page label leaked into zh-CN output:\n%s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("closing stdout pipe: %v", err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading stdout pipe: %v", err)
	}
	return string(out)
}
