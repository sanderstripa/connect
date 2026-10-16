//go:build !windows

package updates

import "os"

func replaceFile(from, to string) error { return os.Rename(from, to) }
