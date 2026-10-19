package screenshot

import (
	"connect-installer.local/core/core"
	"connect-installer.local/core/internal/adaptertest"
	"testing"
)

func TestWindowsInstallationLifecycle(t *testing.T) {
	adaptertest.Lifecycle(t, "screenshot", "Screenshot.exe", func() core.Adapter { return New() })
}
