package cmd

import (
	"fmt"

	"steam-cli/internal/steam"
	"steam-cli/internal/ui"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user STEAMID64|VANITY|PROFILE_URL",
	Short: "Show public Steam Community profile information",
	Args:  cobra.ExactArgs(1),
	RunE: runCommand(func(cmd *cobra.Command, args []string) (any, error) {
		profile, err := client().UserProfile(args[0])
		if err != nil {
			return nil, err
		}
		return profile, nil
	}, func(value any) error {
		profile := value.(*steam.UserProfile)
		fmt.Println(ui.Title.Render(profile.SteamID))
		fmt.Println(ui.Table(
			[]string{"SteamID64", "State", "Privacy", "Visibility", "VAC", "Trade Ban", "Limited"},
			[][]string{{
				profile.SteamID64,
				empty(profile.OnlineState),
				empty(profile.PrivacyState),
				fmt.Sprintf("%d", profile.VisibilityState),
				yesNo(profile.VACBanned != 0),
				empty(profile.TradeBanState),
				yesNo(profile.IsLimitedAccount != 0),
			}},
		))
		if profile.RealName != "" || profile.Location != "" || profile.MemberSince != "" {
			fmt.Println()
			fmt.Println(ui.Accent.Render("Profile"))
			if profile.RealName != "" {
				fmt.Println("Real name: " + profile.RealName)
			}
			if profile.Location != "" {
				fmt.Println("Location: " + profile.Location)
			}
			if profile.MemberSince != "" {
				fmt.Println("Member since: " + profile.MemberSince)
			}
		}
		fmt.Println()
		fmt.Println("Profile URL: https://steamcommunity.com/profiles/" + profile.SteamID64)
		if profile.AvatarFull != "" {
			fmt.Println("Avatar: " + profile.AvatarFull)
		}
		if profile.Summary != "" {
			fmt.Println()
			fmt.Println(ui.Accent.Render("Summary: ") + truncate(profile.Summary, 200))
		}
		return nil
	}),
}
