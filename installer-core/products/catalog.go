package products

import (
	"connect-installer.local/core/core"
	"embed"
	"encoding/json"
	"io/fs"
)

// VersionOverride is set by the shared build tool for the selected product.
var VersionOverride string

func Version(id string) string { return Configure(id, "", nil).Version }

//go:embed *.json
var catalog embed.FS

//go:embed vpn.png
var vpnIcon []byte

func Configure(id, path string, ribbon fs.FS) core.Config {
	raw, err := catalog.ReadFile(id + ".json")
	if err != nil {
		panic(err)
	}
	var c core.Config
	if err = json.Unmarshal(raw, &c); err != nil {
		panic(err)
	}
	if VersionOverride != "" {
		c.Version = VersionOverride
	}
	c.Path = path
	c.Ribbon = ribbon
	if id == "connect" {
		c.Icon = vpnIcon
	}
	return c
}
