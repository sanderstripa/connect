package vpn

import (
	"connect-installer.local/core/core"
	"connect-installer.local/core/internal/adaptertest"
	"testing"
)

func TestWindowsInstallationLifecycle(t *testing.T) {
	adaptertest.Lifecycle(t, "connect", "Connect.exe", func() core.Adapter { return New() })
}
