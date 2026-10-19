package main

import (
	"github.com/tc-hib/winres"
	"os"
)

func main() {
	if len(os.Args) != 4 {
		panic("resources ICON MANIFEST OUT")
	}
	var rs winres.ResourceSet
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	icon, err := winres.LoadICO(f)
	f.Close()
	if err != nil {
		panic(err)
	}
	if err = rs.SetIcon(winres.RT_ICON, icon); err != nil {
		panic(err)
	}
	raw, err := os.ReadFile(os.Args[2])
	if err != nil {
		panic(err)
	}
	manifest, err := winres.AppManifestFromXML(raw)
	if err != nil {
		panic(err)
	}
	rs.SetManifest(manifest)
	out, err := os.Create(os.Args[3])
	if err != nil {
		panic(err)
	}
	if err = rs.WriteObject(out, winres.ArchAMD64); err != nil {
		panic(err)
	}
	if err = out.Close(); err != nil {
		panic(err)
	}
}
