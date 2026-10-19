//go:build windows

package core

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
)

func ValidateDir(dir string) (string, error) {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if system := os.Getenv("WINDIR"); system != "" && (strings.EqualFold(dir, filepath.Clean(system)) || strings.HasPrefix(strings.ToLower(dir), strings.ToLower(filepath.Clean(system))+string(os.PathSeparator))) {
		return "", fmt.Errorf("Windows system directories cannot be used for installation")
	}
	if !filepath.IsAbs(dir) || strings.HasPrefix(dir, `\\`) || filepath.Dir(dir) == dir || strings.ContainsAny(dir, `<>"|?*`) {
		return "", fmt.Errorf("choose an absolute dedicated application folder")
	}
	if strings.Contains(strings.TrimPrefix(dir, filepath.VolumeName(dir)), ":") {
		return "", fmt.Errorf("invalid installation directory")
	}
	for _, p := range []string{os.Getenv("WINDIR"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramData"), os.Getenv("USERPROFILE"), os.Getenv("LOCALAPPDATA"), os.Getenv("APPDATA")} {
		if p != "" && strings.EqualFold(dir, filepath.Clean(p)) {
			return "", fmt.Errorf("choose a dedicated application folder")
		}
	}
	for p := dir; filepath.Dir(p) != p; p = filepath.Dir(p) {
		base := filepath.Base(p)
		trim := strings.TrimRight(base, ". ")
		stem := strings.ToUpper(strings.SplitN(trim, ".", 2)[0])
		if base != trim || stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '0' && stem[3] <= '9' {
			return "", fmt.Errorf("invalid folder name")
		}
		st, err := os.Lstat(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		attrs, e := windows.GetFileAttributes(windows.StringToUTF16Ptr(p))
		if e != nil || !st.IsDir() || attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return "", fmt.Errorf("directory must not contain junctions or symbolic links")
		}
	}
	return dir, nil
}

func OwnedPath(root, relative string) (string, error) {
	rel := filepath.FromSlash(relative)
	if rel == "" || filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" || strings.ContainsAny(rel, `:<>"|?*`) {
		return "", fmt.Errorf("unsafe payload path %q", relative)
	}
	rel = filepath.Clean(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe payload path %q", relative)
	}
	dest := filepath.Join(root, rel)
	base := filepath.Base(dest)
	trim := strings.TrimRight(base, ". ")
	stem := strings.ToUpper(strings.SplitN(trim, ".", 2)[0])
	if base != trim || stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '0' && stem[3] <= '9' {
		return "", fmt.Errorf("invalid payload filename")
	}
	if _, err := ValidateDir(filepath.Dir(dest)); err != nil {
		return "", err
	}
	if attrs, err := windows.GetFileAttributes(windows.StringToUTF16Ptr(dest)); err == nil && attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return "", fmt.Errorf("payload target is a link")
	}
	return dest, nil
}

func CheckClosed(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	h, err := windows.CreateFile(windows.StringToUTF16Ptr(path), windows.DELETE|windows.GENERIC_WRITE, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return fmt.Errorf("close the application before continuing: %w", err)
	}
	return windows.CloseHandle(h)
}
