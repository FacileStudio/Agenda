package users

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersistAvatarFileWritesToStorageDir(t *testing.T) {
	storageDir := t.TempDir()
	service := NewService(nil, storageDir)
	payload := bytes.Repeat([]byte{0x89, 0x50, 0x4e, 0x47}, 256)

	relativePath, absolutePath, err := service.persistAvatarFile(42, bytes.NewReader(payload), "image/png")
	if err != nil {
		t.Fatalf("persist avatar file: %v", err)
	}

	if !strings.HasPrefix(relativePath, "avatars"+string(filepath.Separator)+"user-42-") || !strings.HasSuffix(relativePath, ".png") {
		t.Fatalf("unexpected relative path: %s", relativePath)
	}

	if !strings.HasPrefix(absolutePath, storageDir+string(filepath.Separator)) {
		t.Fatalf("expected absolute path under storage dir, got %s", absolutePath)
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		t.Fatalf("stat avatar file: %v", err)
	}
	if info.Size() != int64(len(payload)) {
		t.Fatalf("unexpected avatar size: got %d want %d", info.Size(), len(payload))
	}
}

func TestRemoveAvatarFileDeletesManagedAvatarOnly(t *testing.T) {
	storageDir := t.TempDir()
	service := NewService(nil, storageDir)

	managedPath := filepath.Join(storageDir, "avatars", "managed.png")
	if err := os.MkdirAll(filepath.Dir(managedPath), 0o755); err != nil {
		t.Fatalf("mkdir managed path: %v", err)
	}
	if err := os.WriteFile(managedPath, []byte("avatar"), 0o644); err != nil {
		t.Fatalf("write managed avatar: %v", err)
	}

	externalPath := filepath.Join(storageDir, "outside.txt")
	if err := os.WriteFile(externalPath, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write external file: %v", err)
	}

	service.removeAvatarFile("/files/avatars/managed.png")
	service.removeAvatarFile("/files/../outside.txt")

	if _, err := os.Stat(managedPath); !os.IsNotExist(err) {
		t.Fatalf("expected managed avatar removed, stat err=%v", err)
	}
	if _, err := os.Stat(externalPath); err != nil {
		t.Fatalf("expected external file preserved, stat err=%v", err)
	}
}
