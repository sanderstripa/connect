package core

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeAdapter struct {
	calls   int
	options Options
	fail    bool
}

func (f *fakeAdapter) Install(o Options) error {
	f.calls++
	f.options = o
	if f.fail {
		return errors.New("disk error")
	}
	return nil
}
func (f *fakeAdapter) Remove() error                      { f.calls++; return nil }
func (f *fakeAdapter) Open() error                        { f.calls++; return nil }
func (f *fakeAdapter) Action(string, []byte) (any, error) { f.calls++; return nil, nil }
func (f *fakeAdapter) Progress() Progress                 { return Progress{Percent: 25} }
func request(h *Host, method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("X-Setup-Token", token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestAuthorizationAndModeBoundaries(t *testing.T) {
	a := &fakeAdapter{}
	h := &Host{adapter: a, token: "secret"}
	for _, p := range []string{"install", "uninstall", "folder", "open", "chrome"} {
		if w := request(h, "POST", "/api/"+p, "{}", "wrong"); w.Code != 403 {
			t.Fatal(p, w.Code)
		}
	}
	if a.calls != 0 {
		t.Fatal("unauthorized request reached adapter")
	}
	for _, p := range []string{"uninstall", "open", "chrome"} {
		if w := request(h, "POST", "/api/"+p, "{}", "secret"); w.Code != 500 {
			t.Fatal(p, w.Code)
		}
	}
	if a.calls != 0 {
		t.Fatal("invalid mode reached adapter")
	}
}
func TestOptionsAndErrorsReachUI(t *testing.T) {
	a := &fakeAdapter{fail: true}
	h := &Host{adapter: a, token: "secret"}
	w := request(h, "POST", "/api/install", `{"path":"C:\\Custom Path\\App","desktop":true,"startMenu":false,"chrome":false,"language":"ru","theme":"dark"}`, "secret")
	if w.Code != 500 || !strings.Contains(w.Body.String(), "disk error") || h.installed {
		t.Fatal(w.Code, w.Body.String())
	}
	if a.calls != 1 || a.options.Path != `C:\Custom Path\App` || !a.options.Desktop || a.options.StartMenu || a.options.Theme != "dark" {
		t.Fatalf("bad options: %+v", a.options)
	}
}
func TestConcurrentOperationDoesNotReachAdapter(t *testing.T) {
	a := &fakeAdapter{}
	h := &Host{adapter: a, token: "secret"}
	h.mu.Lock()
	defer h.mu.Unlock()
	w := request(h, "POST", "/api/install", "{}", "secret")
	if w.Code != 409 || a.calls != 0 {
		t.Fatal(w.Code, a.calls)
	}
}
func TestFolderCancellationKeepsSelectionEmpty(t *testing.T) {
	h := &Host{token: "secret", adapter: &fakeAdapter{}, chooseFolder: func() (string, error) { return "", errors.New("cancelled by user") }}
	w := request(h, "POST", "/api/folder", "{}", "secret")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"cancelled":true`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
