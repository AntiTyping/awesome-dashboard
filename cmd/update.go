package cmd

import (
	"errors"
	"fmt"

	"github.com/AntiTyping/awesome-dashboard/internal/notify"
	"github.com/AntiTyping/awesome-dashboard/internal/store"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <id-prefix>",
	Short: "Update a dashboard item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var title, subtitle, color *string
		for name, dst := range map[string]**string{"title": &title, "subtitle": &subtitle, "color": &color} {
			if cmd.Flags().Changed(name) {
				v, _ := cmd.Flags().GetString(name)
				*dst = &v
			}
		}
		if title == nil && subtitle == nil && color == nil {
			return errors.New("at least one of --title, --subtitle, --color is required")
		}
		if color != nil {
			if err := validateColor(*color); err != nil {
				return err
			}
		}
		st, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer st.Close()

		found, err := st.GetByPrefix(args[0])
		if err != nil {
			return err
		}
		item, err := st.Update(found.ID, title, subtitle, color)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Updated %s  %s  %s  %s\n", item.ID[:8], item.Title, item.Subtitle, item.Color)
		notify.Send(port)
		return nil
	},
}

func init() {
	updateCmd.Flags().String("title", "", "new title")
	updateCmd.Flags().String("subtitle", "", "new subtitle")
	updateCmd.Flags().String("color", "", "new color (#RRGGBB)")
	rootCmd.AddCommand(updateCmd)
}
