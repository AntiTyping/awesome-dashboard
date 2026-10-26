package cmd

import (
	"fmt"
	"regexp"

	"github.com/AntiTyping/awesome-dashboard/internal/notify"
	"github.com/AntiTyping/awesome-dashboard/internal/store"
	"github.com/spf13/cobra"
)

var colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validateColor(c string) error {
	if !colorRe.MatchString(c) {
		return fmt.Errorf("invalid color %q: must match #RRGGBB", c)
	}
	return nil
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a dashboard item",
	RunE: func(cmd *cobra.Command, args []string) error {
		title, _ := cmd.Flags().GetString("title")
		subtitle, _ := cmd.Flags().GetString("subtitle")
		color, _ := cmd.Flags().GetString("color")
		if err := validateColor(color); err != nil {
			return err
		}
		st, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer st.Close()

		item, err := st.Add(title, subtitle, color)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Added %s  %s  %s  %s\n", item.ID[:8], item.Title, item.Subtitle, item.Color)
		notify.Send(port)
		return nil
	},
}

func init() {
	addCmd.Flags().String("title", "", "item title")
	addCmd.Flags().String("subtitle", "", "item subtitle")
	addCmd.Flags().String("color", "", "item color (#RRGGBB)")
	addCmd.MarkFlagRequired("title")
	addCmd.MarkFlagRequired("subtitle")
	addCmd.MarkFlagRequired("color")
	rootCmd.AddCommand(addCmd)
}
