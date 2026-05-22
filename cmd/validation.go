package cmd

import (
	"fmt"
	"strings"

	"steam-cli/internal/steam"

	"github.com/spf13/cobra"
)

func exactArgsWithExample(n int, usage, example string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == n {
			return nil
		}
		return &steam.Error{
			Code:    steam.CodeInvalidInput,
			HintKey: "hint.missing_argument",
			Message: fmt.Sprintf(
				"%s requires %d argument(s), received %d\n\nUsage:\n  %s\n\nExample:\n  %s",
				cmd.CommandPath(),
				n,
				len(args),
				usage,
				example,
			),
		}
	}
}

func validateEnumFlag(name, value string, allowed ...string) error {
	for _, item := range allowed {
		if value == item {
			return nil
		}
	}
	return &steam.Error{
		Code:    steam.CodeInvalidInput,
		HintKey: "hint.invalid_enum",
		Message: fmt.Sprintf(
			"invalid --%s %q; expected one of: %s",
			name,
			value,
			strings.Join(allowed, ", "),
		),
	}
}
