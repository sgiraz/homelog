package database

import (
	"path/filepath"
	"testing"
)

// Uploads and avatars must follow DB_PATH, wherever it points.
func TestDataDir_FollowsDBPath(t *testing.T) {
	dbPath := filepath.Join("srv", "homelog", "homelog.db")
	t.Setenv("DB_PATH", dbPath)
	if got, want := DataDir(), filepath.Join("srv", "homelog"); got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}
}

func TestDataDir_DefaultsNextToDefaultDatabase(t *testing.T) {
	t.Setenv("DB_PATH", "")
	if got, want := DataDir(), filepath.Dir("./data/homelog.db"); got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}
}
