package cmd

import (
	"fmt"
	"net/http"

	"github.com/AntiTyping/awesome-dashboard/internal/server"
	"github.com/AntiTyping/awesome-dashboard/internal/store"
	"github.com/AntiTyping/awesome-dashboard/web"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the dashboard web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer st.Close()

		srv := server.New(st)
		mux := http.NewServeMux()
		srv.RegisterRoutes(mux)
		mux.Handle("GET /{$}", http.FileServer(http.FS(web.Content)))

		fmt.Printf("Dashboard running at http://localhost:%d\n", port)
		return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
