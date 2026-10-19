//go:build windows

package core

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Cleanup only a named uninstaller and the now-empty containing folder.
func CleanupUninstaller(root, name string) error {
	allowed := map[string]bool{"Uninstall.exe": true, "Screencast-Uninstall.exe": true, "Uninstall Soulu.exe": true}
	if !allowed[name] {
		return fmt.Errorf("unknown uninstaller")
	}
	target, err := OwnedPath(root, name)
	if err != nil {
		return err
	}
	script := `Wait-Process -Id ` + strconv.Itoa(os.Getpid()) + ` -ErrorAction SilentlyContinue;Start-Sleep -Milliseconds 300;if(Test-Path -LiteralPath ` + QuotePS(target) + `){Remove-Item -LiteralPath ` + QuotePS(target) + ` -Force};if((Test-Path -LiteralPath ` + QuotePS(root) + `) -and @(Get-ChildItem -LiteralPath ` + QuotePS(root) + ` -Force).Count -eq 0){Remove-Item -LiteralPath ` + QuotePS(root) + `};`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

func QuotePS(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func PowerShell(script string) error {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "$ErrorActionPreference='Stop';"+script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}
func AtomicWrite(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".connect-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}

type FileManifest struct {
	Product string   `json:"product"`
	Files   []string `json:"files"`
}

const ManifestName = "connect-installed-files.json"

// Unpack stages and validates the complete archive before changing destination files.
// It retains backups through the last file replacement and rolls back copy failures.
func Unpack(raw []byte, root, product, executable string, progress func(int)) error {
	if _, err := ValidateDir(root); err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return err
	}
	stage, err := os.MkdirTemp("", "connect-payload-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	m := FileManifest{Product: product}
	seen := map[string]bool{}
	var size uint64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive contains link")
		}
		key := strings.ToLower(filepath.Clean(filepath.FromSlash(f.Name)))
		if seen[key] || key == ManifestName {
			return fmt.Errorf("duplicate or reserved payload path")
		}
		seen[key] = true
		target, e := OwnedPath(stage, f.Name)
		if e != nil {
			return e
		}
		if _, e = OwnedPath(root, f.Name); e != nil {
			return e
		}
		size += f.UncompressedSize64
		if size > 4<<30 {
			return fmt.Errorf("payload size exceeds limit")
		}
		if e = os.MkdirAll(filepath.Dir(target), 0755); e != nil {
			return e
		}
		in, e := f.Open()
		if e != nil {
			return e
		}
		out, e := os.Create(target)
		if e != nil {
			in.Close()
			return e
		}
		_, e = io.Copy(out, in)
		ce := out.Close()
		in.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		m.Files = append(m.Files, f.Name)
	}
	if !seen[strings.ToLower(executable)] {
		return fmt.Errorf("application missing from archive")
	}
	for _, name := range m.Files {
		p, e := OwnedPath(root, name)
		if e != nil {
			return e
		}
		if e = CheckClosed(p); e != nil {
			return e
		}
	}
	if err = os.MkdirAll(root, 0755); err != nil {
		return err
	}
	type backup struct {
		path   string
		data   []byte
		exists bool
	}
	backups := []backup{}
	rollback := func() {
		for i := len(backups) - 1; i >= 0; i-- {
			b := backups[i]
			if b.exists {
				AtomicWrite(b.path, b.data)
			} else {
				os.Remove(b.path)
			}
		}
	}
	for i, name := range m.Files {
		path, e := OwnedPath(root, name)
		if e != nil {
			rollback()
			return e
		}
		old, e := os.ReadFile(path)
		if e != nil && !os.IsNotExist(e) {
			rollback()
			return e
		}
		backups = append(backups, backup{path: path, data: old, exists: e == nil})
		data, e := os.ReadFile(filepath.Join(stage, filepath.FromSlash(name)))
		if e == nil {
			e = os.MkdirAll(filepath.Dir(path), 0755)
		}
		if e == nil {
			e = AtomicWrite(path, data)
		}
		if e != nil {
			rollback()
			return e
		}
		progress(20 + (i+1)*55/len(m.Files))
	}
	if old, e := os.ReadFile(filepath.Join(root, ManifestName)); e == nil {
		var previous FileManifest
		if json.Unmarshal(old, &previous) == nil && previous.Product == product {
			for _, name := range previous.Files {
				if !seen[strings.ToLower(filepath.Clean(filepath.FromSlash(name)))] {
					if _, e := OwnedPath(root, name); e != nil {
						rollback()
						return e
					}
					m.Files = append(m.Files, name)
				}
			}
		}
	}
	blob, _ := json.MarshalIndent(m, "", "  ")
	if err = AtomicWrite(filepath.Join(root, ManifestName), blob); err != nil {
		rollback()
		return err
	}
	return nil
}

func RemoveOwned(root, product, executable string) error {
	if _, err := ValidateDir(root); err != nil {
		return err
	}
	if err := CheckClosed(filepath.Join(root, executable)); err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(root, ManifestName))
	if err != nil {
		return err
	}
	var m FileManifest
	if err = json.Unmarshal(raw, &m); err != nil || m.Product != product {
		return fmt.Errorf("invalid ownership manifest")
	}
	paths := []string{}
	self, _ := os.Executable()
	for _, name := range m.Files {
		p, e := OwnedPath(root, name)
		if e != nil {
			return e
		}
		if strings.EqualFold(p, self) {
			continue
		}
		if e = CheckClosed(p); e != nil {
			return e
		}
		paths = append(paths, p)
	}
	for _, p := range paths {
		if e := os.Remove(p); e != nil && !os.IsNotExist(e) {
			return e
		}
	}
	return os.Remove(filepath.Join(root, ManifestName))
}

func Shortcuts(name, dir, exe, icon, args string, desktop, menu bool) error {
	script := `$w=New-Object -ComObject WScript.Shell;`
	for _, s := range []struct {
		enabled bool
		folder  string
	}{{desktop, "Desktop"}, {menu, "Programs"}} {
		suffix := `\` + name + `.lnk`
		if s.folder == "Programs" {
			suffix = `\` + name + `\` + name + `.lnk`
		}
		script += `$p=[Environment]::GetFolderPath('` + s.folder + `')+` + QuotePS(suffix) + `;`
		if s.enabled {
			script += `[IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($p))|Out-Null;$l=$w.CreateShortcut($p);$l.TargetPath=` + QuotePS(filepath.Join(dir, exe)) + `;$l.Arguments=` + QuotePS(args) + `;$l.WorkingDirectory=` + QuotePS(dir) + `;$l.IconLocation=` + QuotePS(filepath.Join(dir, icon)+",0") + `;$l.Save();`
		} else {
			script += `if(Test-Path -LiteralPath $p){Remove-Item -LiteralPath $p};`
		}
	}
	return PowerShell(script)
}
