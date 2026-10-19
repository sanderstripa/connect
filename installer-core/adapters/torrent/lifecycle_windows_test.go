package torrent

import (
	"connect-installer.local/core/core"
	"connect-installer.local/core/internal/adaptertest"
	"testing"
)

func TestWindowsInstallationLifecycle(t *testing.T) {
	adaptertest.Lifecycle(t, "torrent", "Torrent.exe", func() core.Adapter { return New() })
}
