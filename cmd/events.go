package cmd

import (
	"fmt"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

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

		rows := make([][]string, 0, len(events))
		for _, event := range events {
			rows = append(rows, []string{
				event.StartDate,
				event.EndDate,
				event.Status,
				event.Category,
				event.Name,
				truncate(event.Description, 56),
			})
		}
		if len(rows) == 0 {
			fmt.Println(ui.Muted.Render("No events found."))
			return nil
		}
		fmt.Println(ui.Table([]string{"Start", "End", "Status", "Category", "Event", "Description"}, rows))
		return nil
	}),
}

func init() {
	eventsCmd.Flags().IntVar(&eventOpts.pastDays, "past-days", 45, "include events that ended within this many days")
	eventsCmd.Flags().IntVar(&eventOpts.futureDays, "future-days", 180, "include events starting within this many days")
}
