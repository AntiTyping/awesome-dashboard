package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AntiTyping/awesome-dashboard/internal/store"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func resetFlags(c *cobra.Command) {
	c.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

func setupTestCmd(t *testing.T) (*cobra.Command, string) {
	t.Helper()
	db := filepath.Join(t.TempDir(), "test.db")
	resetFlags(rootCmd)
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	t.Cleanup(func() { rootCmd.SetArgs(nil); rootCmd.SetOut(nil) })
	return rootCmd, db
}

func run(t *testing.T, root *cobra.Command, db string, args ...string) (string, error) {
	t.Helper()
	resetFlags(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(append(args, "--db", db, "--port", "1"))
	err := root.Execute()
	return buf.String(), err
}

func seed(t *testing.T, db, title, sub, color string) *store.Item {
	t.Helper()
	st, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	it, err := st.Add(title, sub, color)
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func getAll(t *testing.T, db string) []store.Item {
	t.Helper()
	st, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	items, err := st.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func TestAddCommand(t *testing.T) {
	root, db := setupTestCmd(t)
	out, err := run(t, root, db, "add", "--title", "Project A", "--subtitle", "In progress", "--color", "#FF5733")
	if err != nil {
		t.Fatal(err)
	}
	items := getAll(t, db)
	if len(items) != 1 || items[0].Title != "Project A" || items[0].Subtitle != "In progress" || items[0].Color != "#FF5733" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if !strings.Contains(out, "Project A") || !strings.Contains(out, items[0].ID[:8]) {
		t.Fatalf("output missing title/id prefix: %q", out)
	}
}

func TestAddCommand_InvalidColor(t *testing.T) {
	root, db := setupTestCmd(t)
	_, err := run(t, root, db, "add", "--title", "X", "--subtitle", "Y", "--color", "red")
	if err == nil || !strings.Contains(err.Error(), "invalid color") {
		t.Fatalf("expected invalid color error, got %v", err)
	}
}

func TestAddCommand_MissingRequiredFlag(t *testing.T) {
	root, db := setupTestCmd(t)
	if _, err := run(t, root, db, "add", "--title", "X", "--subtitle", "Y"); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateCommand(t *testing.T) {
	root, db := setupTestCmd(t)
	it := seed(t, db, "Old", "Sub", "#112233")
	out, err := run(t, root, db, "update", it.ID[:8], "--title", "New Title")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "New Title") {
		t.Fatalf("output: %q", out)
	}
	items := getAll(t, db)
	if items[0].Title != "New Title" || items[0].Subtitle != "Sub" || items[0].Color != "#112233" {
		t.Fatalf("unexpected: %+v", items[0])
	}
}

func TestUpdateCommand_NoFlags(t *testing.T) {
	root, db := setupTestCmd(t)
	it := seed(t, db, "Old", "Sub", "#112233")
	_, err := run(t, root, db, "update", it.ID[:8])
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected at least one error, got %v", err)
	}
}

func TestRemoveCommand(t *testing.T) {
	root, db := setupTestCmd(t)
	it := seed(t, db, "Gone", "Sub", "#112233")
	out, err := run(t, root, db, "remove", it.ID[:8])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Removed") || !strings.Contains(out, "Gone") {
		t.Fatalf("output: %q", out)
	}
	if n := len(getAll(t, db)); n != 0 {
		t.Fatalf("expected empty, got %d", n)
	}
}

func TestListCommand_Empty(t *testing.T) {
	root, db := setupTestCmd(t)
	out, err := run(t, root, db, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No items") {
		t.Fatalf("output: %q", out)
	}
}

func TestListCommand_WithItems(t *testing.T) {
	root, db := setupTestCmd(t)
	a := seed(t, db, "Alpha", "S1", "#111111")
	b := seed(t, db, "Beta", "S2", "#222222")
	out, err := run(t, root, db, "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"Alpha", "Beta", a.ID[:8], b.ID[:8]} {
		if !strings.Contains(out, s) {
			t.Fatalf("output missing %q: %q", s, out)
		}
	}
}
