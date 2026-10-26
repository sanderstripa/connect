package products

import (
	"bytes"
	"image/png"
	"testing"
)

func TestVPNWindowUsesProductIcon(t *testing.T) {
	c := Configure("connect", "", nil)
	image, err := png.Decode(bytes.NewReader(c.Icon))
	if err != nil {
		t.Fatal(err)
	}
	if image.Bounds().Dx() != 256 || image.Bounds().Dy() != 256 {
		t.Fatal("VPN window icon must be 256px")
	}
	if _, _, _, a := image.At(0, 0).RGBA(); a != 0 {
		t.Fatal("VPN window icon must be transparent outside the circle")
	}
	if len(Configure("screenshot", "", nil).Icon) != 0 {
		t.Fatal("VPN icon leaked to another product")
	}
}
