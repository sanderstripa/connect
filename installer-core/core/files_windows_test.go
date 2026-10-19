package core

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func archive(files map[string]string) []byte {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for name, data := range files {
		f, _ := z.Create(name)
		f.Write([]byte(data))
	}
	z.Close()
	return b.Bytes()
}
func TestTraversalFailsBeforeChangingExistingFiles(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "App.exe"), []byte("old"), 0600)
	err := Unpack(archive(map[string]string{"App.exe": "new", "../foreign.txt": "bad"}), root, "test", "App.exe", func(int) {})
	if err == nil {
		t.Fatal("accepted traversal")
	}
	raw, _ := os.ReadFile(filepath.Join(root, "App.exe"))
	if string(raw) != "old" {
		t.Fatal("changed existing files before validation")
	}
}
func TestManifestRemovalPreservesUnrelatedFiles(t *testing.T) {
	root := t.TempDir()
	sentinel := filepath.Join(root, "personal.txt")
	os.WriteFile(sentinel, []byte("keep"), 0600)
	for _, data := range []string{"version1", "version2"} {
		if err := Unpack(archive(map[string]string{"App.exe": data, "assets/model.bin": "model"}), root, "test", "App.exe", func(int) {}); err != nil {
			t.Fatal(err)
		}
	}
	if err := RemoveOwned(root, "test", "App.exe"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(sentinel)
	if err != nil || string(raw) != "keep" {
		t.Fatal("unrelated data lost")
	}
	if _, err = os.Stat(filepath.Join(root, "App.exe")); !os.IsNotExist(err) {
		t.Fatal("owned application remains")
	}
}
func TestRejectUnsafePaths(t *testing.T) {
	for _, p := range []string{`C:\`, `relative`, `C:\Users\CON\app`, `C:\App.\Child`, `C:\App:stream`, `\\server\share\app`} {
		if _, err := ValidateDir(p); err == nil {
			t.Fatal("accepted", p)
		}
	}
}
