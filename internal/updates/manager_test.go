package updates

import (
	"archive/zip"
	"bytes"
	"debug/buildinfo"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func zipFiles(t *testing.T, names []string, data []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, name := range names {
		w, e := z.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		w.Write(data)
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func fixture(t *testing.T) (*Manager, *[]githubRelease, *map[string][]byte) {
	t.Helper()
	releases := []githubRelease{}
	files := map[string][]byte{}
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if data, ok := files[r.URL.Path]; ok {
			w.Write(data)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	t.Cleanup(server.Close)
	m := New(t.TempDir(), map[string]Baseline{})
	m.api = server.URL
	m.client = server.Client()
	return m, &releases, &files
}
func executable(t *testing.T) []byte {
	t.Helper()
	file, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(file)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestInstallSurvivesRestartRollbackAndTamperDetection(t *testing.T) {
	m, releases, files := fixture(t)
	exe := executable(t)
	if verifyExecutable(exe) != nil {
		t.Skip("Windows executable test")
	}
	baseline := filepath.Join(t.TempDir(), "xray.exe")
	os.WriteFile(baseline, exe, 0600)
	m.Baselines["xray"] = Baseline{Version: "v1", Files: map[string]string{"xray.exe": baseline}}
	archive := zipFiles(t, []string{"nested/xray.exe"}, exe)
	(*files)["/x.zip"] = archive
	*releases = []githubRelease{{Tag: "v2-preview", Prerelease: true, Assets: []asset{{Name: "Xray-windows-64.zip", URL: m.api + "/x.zip", Size: int64(len(archive)), Digest: "sha256:" + hash(archive)}}}}
	stable := m.Status("xray", true, false)
	if len(stable.Releases) != 0 {
		t.Fatal("prerelease leaked into stable catalog")
	}
	preview := m.Status("xray", true, true)
	if len(preview.Releases) != 1 || !preview.Releases[0].Prerelease {
		t.Fatal("prerelease missing")
	}
	if _, e := m.Install("xray", "v2-preview"); e != nil {
		t.Fatal(e)
	}
	status := m.Status("xray", false, true)
	if status.Installed != "v2-preview" || status.Previous != "v1" {
		t.Fatalf("wrong versions: %+v", status)
	}
	path, e := m.Resolve("xray", "xray.exe", baseline)
	if e != nil || path == baseline {
		t.Fatal("new slot not active")
	}
	reloaded := New(m.Root, m.Baselines)
	if result, e := reloaded.Rollback("xray"); e != nil || result.Version != "v1" {
		t.Fatal("rollback after restart failed", e)
	}
	old, e := reloaded.Resolve("xray", "xray.exe", baseline)
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(old, []byte("damaged"), 0600)
	if _, e = reloaded.Resolve("xray", "xray.exe", baseline); e == nil {
		t.Fatal("tampered binary accepted")
	}
	original, _ := os.ReadFile(baseline)
	if !bytes.Equal(original, exe) {
		t.Fatal("bundled baseline modified")
	}
}
func TestChecksumFailureDoesNotChangeActiveSlot(t *testing.T) {
	m, releases, files := fixture(t)
	data := bytes.Repeat([]byte{1}, 64)
	(*files)["/geoip"] = data
	(*files)["/geosite"] = data
	*releases = []githubRelease{{Tag: "20261017", Assets: []asset{{Name: "geoip.dat", URL: m.api + "/geoip", Digest: "sha256:" + hash(data), Size: int64(len(data))}, {Name: "geosite.dat", URL: m.api + "/geosite", Digest: "sha256:" + hash([]byte("wrong")), Size: int64(len(data))}}}}
	if _, e := m.Install("geo", "20261017"); e == nil {
		t.Fatal("bad digest accepted")
	}
	s, e := m.readState("geo")
	if e != nil || s.Current != nil {
		t.Fatal("partial Geo update became active")
	}
}
func TestArchiveSelectionAndUntrustedDownloads(t *testing.T) {
	data := []byte("test")
	files, e := extract(zipFiles(t, []string{"../../xray.exe"}, data), []string{"xray.exe"})
	if e != nil || !bytes.Equal(files["xray.exe"], data) {
		t.Fatal("basename extraction failed")
	}
	if _, e = extract(zipFiles(t, []string{"xray.exe", "sub/XRAY.EXE"}, data), []string{"xray.exe"}); e == nil {
		t.Fatal("duplicate executable accepted")
	}
	if _, e = extract(zipFiles(t, []string{"other.exe"}, data), []string{"xray.exe"}); e == nil {
		t.Fatal("missing executable accepted")
	}
	m := New(t.TempDir(), nil)
	for _, raw := range []string{"http://github.com/a", "https://evil.example/a", "file:///tmp/a", "https://name:password@github.com/a"} {
		if m.allowed(raw) == nil {
			t.Fatal("unsafe URL accepted")
		}
	}
	if validDigest("sha256:123") || verifyExecutable(data) == nil {
		t.Fatal("invalid content accepted")
	}
}

func TestGeoFirstInstallCanRollBackToNotInstalled(t *testing.T) {
	m, releases, files := fixture(t)
	m.Baselines["geo"] = Baseline{Files: map[string]string{"geoip.dat": filepath.Join(t.TempDir(), "geoip.dat"), "geosite.dat": filepath.Join(t.TempDir(), "geosite.dat")}}
	data := bytes.Repeat([]byte{1}, 64)
	(*files)["/geoip"] = data
	(*files)["/geosite"] = data
	*releases = []githubRelease{{Tag: "20261017", Assets: []asset{{Name: "geoip.dat", URL: m.api + "/geoip", Digest: "sha256:" + hash(data), Size: int64(len(data))}, {Name: "geosite.dat", URL: m.api + "/geosite", Digest: "sha256:" + hash(data), Size: int64(len(data))}}}}
	if _, e := m.Install("geo", "20261017"); e != nil {
		t.Fatal(e)
	}
	if m.Status("geo", false, true).Previous != "Not installed" {
		t.Fatal("missing rollback to absent databases")
	}
	if _, e := m.Rollback("geo"); e != nil {
		t.Fatal(e)
	}
	if m.Status("geo", false, true).Installed != "" {
		t.Fatal("databases still active after rollback")
	}
	file, e := m.Resolve("geo", "geoip.dat", "no-geodata")
	if e != nil || file != "no-geodata" {
		t.Fatal("rollback to baseline failed")
	}
}

func TestApplicationRollbackUsesCachedInstallerOffline(t *testing.T) {
	m, releases, files := fixture(t)
	exe := executable(t)
	if verifyExecutable(exe) != nil {
		t.Skip("Windows executable test")
	}
	baseline := filepath.Join(t.TempDir(), "Uninstall.exe")
	os.WriteFile(baseline, exe, 0600)
	m.Baselines["app"] = Baseline{Version: "1.0.0", Executable: baseline, Files: map[string]string{"Connect-Setup.exe": baseline}}
	(*files)["/setup.exe"] = exe
	*releases = []githubRelease{{Tag: "v2.0.0", Assets: []asset{{Name: "Connect-Setup.exe", URL: m.api + "/setup.exe", Size: int64(len(exe)), Digest: "sha256:" + hash(exe)}}}}
	result, e := m.Install("app", "v2.0.0")
	if e != nil || result.Installer == "" {
		t.Fatal("application staging failed", e)
	}
	if m.Status("app", false, true).Installed != "1.0.0" {
		t.Fatal("staged installer was reported as running application")
	}
	m.Baselines["app"] = Baseline{Version: "2.0.0"}
	reopened := New(m.Root, m.Baselines)
	if reopened.Status("app", false, true).Previous != "1.0.0" {
		t.Fatal("previous installer not recorded")
	}
	// New manager has no test HTTP server configured: rollback must stay local.
	result, e = reopened.Rollback("app")
	if e != nil || result.Installer == "" || result.Version != "1.0.0" {
		t.Fatal("offline app rollback failed", e)
	}
	if reopened.Status("app", false, true).Previous != "1.0.0" {
		t.Fatal("failed or cancelled installer lost offline rollback")
	}
	if retry, e := reopened.Rollback("app"); e != nil || retry.Version != "1.0.0" {
		t.Fatal("cannot retry offline rollback", e)
	}
	saved, e := os.ReadFile(result.Installer)
	if e != nil || !bytes.Equal(saved, exe) {
		t.Fatal("wrong cached installer")
	}
}

func TestSudokuCatalogKeepsProtocolAndAdapterBuildVersions(t *testing.T) {
	m, releases, _ := fixture(t)
	*releases = []githubRelease{
		{Tag: "v0.5.2-1", Assets: []asset{{Name: "sudoku-windows-amd64.zip", Digest: "sha256:" + hash([]byte("official"))}}},
		{Tag: "v0.3.2", Assets: []asset{{Name: "Connect-Sudoku-v0.5.2-1-windows-amd64.zip", Digest: "sha256:" + hash([]byte("compatible"))}}},
	}
	status := m.Status("sudoku", true, false)
	if status.Error != "" || status.Source != "sanderstripa/connect" || len(status.Releases) != 1 || status.Releases[0].Version != "v0.5.2-1+connect.0.3.2" {
		t.Fatalf("unexpected compatible catalog: %+v", status)
	}
}

func TestOfficialSudokuSlotFallsBackWithoutDeletingItsFiles(t *testing.T) {
	m, _, _ := fixture(t)
	exe := executable(t)
	if verifyExecutable(exe) != nil {
		t.Skip("Windows executable fixture")
	}
	g, err := m.store("sudoku", "v0.5.2-1", map[string][]byte{"sudoku.exe": exe})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.save("sudoku", state{Current: g}); err != nil {
		t.Fatal(err)
	}
	baseline := filepath.Join(t.TempDir(), "bundled-sudoku.exe")
	m.Baselines["sudoku"] = Baseline{Version: "v0.5.2-1+connect.0.3.2", Files: map[string]string{"sudoku.exe": baseline}}
	path, err := m.Resolve("sudoku", "sudoku.exe", baseline)
	if err != nil || path != baseline {
		t.Fatal("incompatible slot overrode compatible baseline", path, err)
	}
	if got := m.Status("sudoku", false, false).Installed; got != m.Baselines["sudoku"].Version {
		t.Fatal("wrong active version", got)
	}
	dir, _ := m.slot("sudoku", g)
	if _, err = os.Stat(filepath.Join(dir, "sudoku.exe")); err != nil {
		t.Fatal("previous binary was removed", err)
	}
}

func TestBundledSudokuHasRequiredDesktopAdapters(t *testing.T) {
	path := filepath.Join("..", "..", "runtime", "bin", "windows-amd64", "sudoku.exe")
	_, statErr := os.Stat(path)
	if os.IsNotExist(statErr) {
		t.Skip("bundled Windows runtime not built on this host")
	}
	info, err := buildinfo.ReadFile(path)
	if err != nil || !compatibleSudokuBuild(info) {
		t.Fatal("built core lacks desktop adapters or MPTCP mitigation", err)
	}
}
