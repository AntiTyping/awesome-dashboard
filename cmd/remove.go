package cmd

import (
	"fmt"

	"github.com/AntiTyping/awesome-dashboard/internal/notify"
	"github.com/AntiTyping/awesome-dashboard/internal/store"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <id-prefix>",
	Short: "Remove a dashboard item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer st.Close()

		item, err := st.GetByPrefix(args[0])
		if err != nil {
			return err
		}
		if err := st.Delete(item.ID); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Removed %s  %s\n", item.ID[:8], item.Title)
		notify.Send(port)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
