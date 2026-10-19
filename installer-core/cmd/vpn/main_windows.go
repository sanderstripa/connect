package main

import (
	"connect-installer.local/core/adapters/vpn"
	"connect-installer.local/core/core"
	"connect-installer.local/core/products"
)

func main() {
	a := vpn.New()
	if err := core.Run(products.Configure("connect", a.Path(), nil), a); err != nil {
		panic(err)
	}
}
