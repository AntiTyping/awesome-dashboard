package store

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpen_CreatesSchema(t *testing.T) {
	s := newStore(t)
	var name string
	err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='items'`).Scan(&name)
	if err != nil || name != "items" {
		t.Fatalf("items table missing: %v", err)
	}
}

func TestAdd_And_GetAll(t *testing.T) {
	s := newStore(t)
	a, err := s.Add("A", "sub a", "red")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := s.Add("B", "sub b", "blue"); err != nil {
		t.Fatal(err)
	}
	items, err := s.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].ID != a.ID || items[0].Title != "A" || items[0].Subtitle != "sub a" || items[0].Color != "red" {
		t.Errorf("bad first item: %+v", items[0])
	}
	if items[1].Title != "B" {
		t.Errorf("order wrong: %+v", items)
	}
	for _, it := range items {
		if _, err := uuid.Parse(it.ID); err != nil {
			t.Errorf("invalid uuid %q", it.ID)
		}
		if it.CreatedAt.IsZero() || it.UpdatedAt.IsZero() {
			t.Errorf("timestamps not set: %+v", it)
		}
	}
}

func TestGetByPrefix_Unique(t *testing.T) {
	s := newStore(t)
	a, _ := s.Add("A", "", "")
	got, err := s.GetByPrefix(a.ID[:8])
	if err != nil || got.ID != a.ID {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func TestGetByPrefix_NotFound(t *testing.T) {
	s := newStore(t)
	if _, err := s.GetByPrefix("nonexist"); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetByPrefix_Ambiguous(t *testing.T) {
	s := newStore(t)
	s.Add("A", "", "")
	s.Add("B", "", "")
	_, err := s.GetByPrefix("")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

func TestUpdate_AllFields(t *testing.T) {
	s := newStore(t)
	a, _ := s.Add("A", "s", "red")
	time.Sleep(1100 * time.Millisecond)
	t2, s2, c2 := "T", "S", "green"
	got, err := s.Update(a.ID, &t2, &s2, &c2)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "T" || got.Subtitle != "S" || got.Color != "green" {
		t.Errorf("bad values: %+v", got)
	}
	if !got.UpdatedAt.After(a.UpdatedAt) {
		t.Errorf("UpdatedAt not changed: %v vs %v", got.UpdatedAt, a.UpdatedAt)
	}
}

func TestUpdate_PartialFields(t *testing.T) {
	s := newStore(t)
	a, _ := s.Add("A", "s", "red")
	nt := "New"
	got, err := s.Update(a.ID, &nt, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "New" || got.Subtitle != "s" || got.Color != "red" {
		t.Errorf("bad values: %+v", got)
	}
}

func TestDelete(t *testing.T) {
	s := newStore(t)
	a, _ := s.Add("A", "", "")
	if err := s.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	items, _ := s.GetAll()
	if len(items) != 0 {
		t.Fatalf("expected empty, got %d", len(items))
	}
}

func TestDelete_NotFound(t *testing.T) {
	s := newStore(t)
	if err := s.Delete("nope"); err == nil {
		t.Fatal("expected error")
	}
}
