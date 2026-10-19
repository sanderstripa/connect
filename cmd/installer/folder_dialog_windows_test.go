//go:build windows && legacy_installer

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

func TestFolderDialogCancellationAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name      string
		path      string
		err       error
		status    int
		cancelled bool
	}{
		{"selected", `D:\Apps\Connect`, nil, 200, false},
		{"empty cancellation", "", nil, 200, true},
		{"Wails cancellation", "", errors.New("cancelled by user"), 200, true},
		{"wrapped Wails cancellation", "", fmt.Errorf("folder: %w", errors.New("cancelled by user")), 200, true},
		{"Windows cancellation", "", windows.ERROR_CANCELLED, 200, true},
		{"COM cancellation", "", ole.NewError(0x800704c7), 200, true},
		{"access denied", "", windows.ERROR_ACCESS_DENIED, 500, false},
		{"COM access denied", "", ole.NewError(0x80070005), 500, false},
		{"other failure", "", errors.New("dialog initialization failed"), 500, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const original = `C:\Program Files\Connect`
			s := &setup{token: "test", dir: original, chooseFolder: func() (string, error) { return tc.path, tc.err }}
			r := httptest.NewRequest("POST", "/api/folder", nil)
			r.Header.Set("X-Setup-Token", "test")
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if tc.status == 200 {
				var got folderSelection
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got.Cancelled != tc.cancelled || got.Path != tc.path {
					t.Fatalf("selection: %+v", got)
				}
			}
			if s.dir != original || s.installed || s.progress.Load() != 0 {
				t.Fatal("folder dialog mutated installation state")
			}
		})
	}
}
