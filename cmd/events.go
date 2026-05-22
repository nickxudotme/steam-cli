package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

var eventOpts struct {
	pastDays   int
	futureDays int
}

var eventsCmd = &cobra.Command{
	Use:     "events",
	Aliases: []string{"fests", "festivals"},
	Short:   "List recent and upcoming Steam sale events",
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		events, err := client().Events(steam.EventQuery{
			PastDays:   eventOpts.pastDays,
			FutureDays: eventOpts.futureDays,
		})
		if err != nil {
			return nil, err
		}
		return events, nil
	}, func(value any) error {
		events := value.([]steam.Event)
		return renderEvents(events)
	}),
}

func renderEvents(events []steam.Event) error {
	descriptionWidth := eventDescriptionWidth(events, terminalWidth())

	rows := make([][]string, 0, len(events))
	for _, event := range events {
		rows = append(rows, []string{
			event.StartDate,
			event.EndDate,
			event.Status,
			event.Category,
			event.Name,
			wrapEventDescription(event.Description, descriptionWidth),
		})
	}
	if len(rows) == 0 {
		fmt.Println(ui.Muted.Render("No events found."))
		return nil
	}
	fmt.Println(ui.TableWithRowBorders([]string{"Start", "End", "Status", "Category", "Event", "Description"}, rows))
	fmt.Println()
	fmt.Println(ui.KeyValue(i18n.T("events.label.official_page"), steam.SteamworksUpcomingEvents))
	return nil
}

func init() {
	eventsCmd.Flags().IntVar(&eventOpts.pastDays, "past-days", 45, "include events that ended within this many days")
	eventsCmd.Flags().IntVar(&eventOpts.futureDays, "future-days", 180, "include events starting within this many days")
}

const (
	defaultTerminalWidth       = 120
	minEventDescriptionWidth   = 32
	eventDescriptionTableCells = 6
)

func terminalWidth() int {
	return terminalWidthFrom(func() (int, error) {
		width, _, err := term.GetSize(os.Stdout.Fd())
		return width, err
	}, os.Getenv)
}

func terminalWidthFrom(getSize func() (int, error), getenv func(string) string) int {
	width, err := getSize()
	if err == nil && width > 0 {
		return width
	}
	if value := strings.TrimSpace(getenv("COLUMNS")); value != "" {
		if columns, err := strconv.Atoi(value); err == nil && columns > 0 {
			return columns
		}
	}
	return defaultTerminalWidth
}

func eventDescriptionWidth(events []steam.Event, terminalWidth int) int {
	widths := []int{
		lipgloss.Width("Start"),
		lipgloss.Width("End"),
		lipgloss.Width("Status"),
		lipgloss.Width("Category"),
		lipgloss.Width("Event"),
	}
	for _, event := range events {
		values := []string{
			event.StartDate,
			event.EndDate,
			event.Status,
			event.Category,
			event.Name,
		}
		for col, value := range values {
			if valueWidth := lipgloss.Width(value); valueWidth > widths[col] {
				widths[col] = valueWidth
			}
		}
	}

	fixedWidth := 0
	for _, width := range widths {
		fixedWidth += width
	}
	tableOverhead := 3*eventDescriptionTableCells + 1
	descriptionWidth := terminalWidth - fixedWidth - tableOverhead
	if descriptionWidth < minEventDescriptionWidth {
		return minEventDescriptionWidth
	}
	return descriptionWidth
}

func wrapEventDescription(value string, width int) string {
	value = strings.Join(strings.Fields(stripTags(value)), " ")
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}

	var lines []string
	var line strings.Builder
	lineWidth := 0
	flush := func() {
		if line.Len() == 0 {
			return
		}
		lines = append(lines, line.String())
		line.Reset()
		lineWidth = 0
	}
	addRune := func(r rune) {
		rw := lipgloss.Width(string(r))
		if lineWidth > 0 && lineWidth+rw > width {
			flush()
		}
		line.WriteRune(r)
		lineWidth += rw
	}

	for _, word := range strings.Fields(value) {
		wordWidth := lipgloss.Width(word)
		if lineWidth > 0 && lineWidth+1+wordWidth <= width {
			line.WriteByte(' ')
			line.WriteString(word)
			lineWidth += 1 + wordWidth
			continue
		}
		if lineWidth > 0 {
			flush()
		}
		if wordWidth <= width {
			line.WriteString(word)
			lineWidth = wordWidth
			continue
		}
		for _, r := range word {
			addRune(r)
		}
	}
	flush()
	return strings.Join(lines, "\n")
}
