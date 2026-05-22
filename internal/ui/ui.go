package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/termenv"
)

var (
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	Muted = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	Accent = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("81"))

	Good = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	Warn = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("238")).
		Padding(0, 1)

	Cell = lipgloss.NewStyle().
		Padding(0, 1)

	SectionTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("81"))

	Label = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("245"))
)

func DisableColor() {
	lipgloss.SetColorProfile(termenv.Ascii)
}

func Section(title string) string {
	return SectionTitle.Render(title)
}

func KeyValue(label string, value string) string {
	if strings.TrimSpace(value) == "" {
		value = "-"
	}
	return Label.Render(label+": ") + value
}

func Table(headers []string, rows [][]string) string {
	return renderTable(headers, rows, false)
}

func TableWithRowBorders(headers []string, rows [][]string) string {
	return renderTable(headers, rows, true)
}

func renderTable(headers []string, rows [][]string, borderRows bool) string {
	t := table.New().
		Headers(headers...).
		Rows(rows...).
		Border(lipgloss.NormalBorder()).
		BorderRow(borderRows).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("238"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return Header
			}
			return Cell
		})
	return t.Render()
}

func Money(cents int, currency string) string {
	if currency == "" {
		return fmt.Sprintf("%.2f", float64(cents)/100)
	}
	return fmt.Sprintf("%.2f %s", float64(cents)/100, currency)
}

func Price(final, initial, discount int, currency string, isFree bool) string {
	if isFree {
		return Good.Render("free")
	}
	if final == 0 && initial == 0 {
		return Muted.Render("unavailable")
	}
	text := Money(final, currency)
	if discount > 0 {
		return fmt.Sprintf("%s %s", Good.Render(text), Warn.Render("-"+strconv.Itoa(discount)+"%"))
	}
	return text
}

func Join(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}
