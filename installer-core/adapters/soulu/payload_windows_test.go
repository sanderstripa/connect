package soulu

import "testing"

func TestBundledCEFIntegrity(t *testing.T) {
	raw, err := assets.ReadFile("payload.zip")
	if err != nil {
		t.Fatal(err)
	}
	if err = verifyCEF(raw); err != nil {
		t.Fatal(err)
	}
}
