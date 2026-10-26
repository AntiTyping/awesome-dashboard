package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	dbPath string
	port   int
)

var rootCmd = &cobra.Command{
	Use:   "awesome-dashboard",
	Short: "A CLI dashboard tool",
}

func defaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "data.db"
	}
	return filepath.Join(home, ".awesome-dashboard", "data.db")
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", defaultDBPath(), "path to SQLite database")
	rootCmd.PersistentFlags().IntVar(&port, "port", 8080, "HTTP server port")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
