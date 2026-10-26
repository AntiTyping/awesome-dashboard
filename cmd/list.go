package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/AntiTyping/awesome-dashboard/internal/store"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List dashboard items",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer st.Close()

		items, err := st.GetAll()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		if len(items) == 0 {
			fmt.Fprintln(out, "No items.")
			return nil
		}
		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tSUBTITLE\tCOLOR")
		for _, it := range items {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", it.ID[:8], it.Title, it.Subtitle, it.Color)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
