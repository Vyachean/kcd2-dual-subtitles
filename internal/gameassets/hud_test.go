package gameassets

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadHUDReadsExactRetailPath(t *testing.T) {
	gameRoot := t.TempDir()
	pakPath := filepath.Join(gameRoot, "Data", "IPL_GameData.pak")
	if err := os.MkdirAll(filepath.Dir(pakPath), 0o755); err != nil {
		t.Fatalf("create Data directory: %v", err)
	}
	writeTestPAK(t, pakPath, map[string][]byte{
		"Libs/UI/hud.gfx":  []byte("hud-bytes"),
		"Libs/UI/Menu.gfx": []byte("menu"),
	})

	got, err := ReadHUD(gameRoot)
	if err != nil {
		t.Fatalf("ReadHUD() error = %v", err)
	}
	if string(got) != "hud-bytes" {
		t.Fatalf("ReadHUD() = %q, want hud-bytes", got)
	}
}

func TestReadHUDAcceptsArchiveCaseAndSlashDifferencesButRejectsDuplicates(t *testing.T) {
	t.Run("case and slash", func(t *testing.T) {
		gameRoot := t.TempDir()
		pakPath := filepath.Join(gameRoot, "Data", "IPL_GameData.pak")
		if err := os.MkdirAll(filepath.Dir(pakPath), 0o755); err != nil {
			t.Fatalf("create Data directory: %v", err)
		}
		writeTestPAK(t, pakPath, map[string][]byte{
			`libs\ui\HUD.GFX`: []byte("hud"),
		})
		got, err := ReadHUD(gameRoot)
		if err != nil || string(got) != "hud" {
			t.Fatalf("ReadHUD() = %q, %v; want hud", got, err)
		}
	})

	t.Run("duplicate semantic path", func(t *testing.T) {
		gameRoot := t.TempDir()
		pakPath := filepath.Join(gameRoot, "Data", "IPL_GameData.pak")
		if err := os.MkdirAll(filepath.Dir(pakPath), 0o755); err != nil {
			t.Fatalf("create Data directory: %v", err)
		}
		writeTestPAKEntries(t, pakPath, []testEntry{
			{name: "Libs/UI/hud.gfx", data: []byte("one")},
			{name: "libs/ui/HUD.GFX", data: []byte("two")},
		})
		_, err := ReadHUD(gameRoot)
		if !errors.Is(err, ErrHUDDuplicate) {
			t.Fatalf("ReadHUD() error = %v, want ErrHUDDuplicate", err)
		}
	})
}

func TestReadHUDFailsClosedForMissingPAKOrHUD(t *testing.T) {
	t.Run("missing pak", func(t *testing.T) {
		_, err := ReadHUD(t.TempDir())
		if err == nil {
			t.Fatal("ReadHUD() error = nil, want error")
		}
	})

	t.Run("missing hud", func(t *testing.T) {
		gameRoot := t.TempDir()
		pakPath := filepath.Join(gameRoot, "Data", "IPL_GameData.pak")
		if err := os.MkdirAll(filepath.Dir(pakPath), 0o755); err != nil {
			t.Fatalf("create Data directory: %v", err)
		}
		writeTestPAK(t, pakPath, map[string][]byte{"Libs/UI/Menu.gfx": []byte("menu")})
		_, err := ReadHUD(gameRoot)
		if !errors.Is(err, ErrHUDNotFound) {
			t.Fatalf("ReadHUD() error = %v, want ErrHUDNotFound", err)
		}
	})
}

type testEntry struct {
	name string
	data []byte
}

func writeTestPAK(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	ordered := make([]testEntry, 0, len(entries))
	for name, data := range entries {
		ordered = append(ordered, testEntry{name: name, data: data})
	}
	writeTestPAKEntries(t, path, ordered)
}

func writeTestPAKEntries(t *testing.T, path string, entries []testEntry) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test PAK: %v", err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		out, err := writer.Create(entry.name)
		if err != nil {
			t.Fatalf("create PAK entry %q: %v", entry.name, err)
		}
		if _, err := out.Write(entry.data); err != nil {
			t.Fatalf("write PAK entry %q: %v", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close PAK writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close PAK file: %v", err)
	}
}
