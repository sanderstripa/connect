// Package updates installs verified components into immutable, per-user slots.
// Changing the active slot never overwrites an executable used by a connection.
package updates

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const AppVersion = "0.1.9"

var operations sync.Mutex

type Baseline struct {
	Executable string
	Version    string
	Build      string
	Files      map[string]string
}
type Release struct {
	Version    string  `json:"version"`
	Prerelease bool    `json:"prerelease"`
	Published  string  `json:"published"`
	Assets     []asset `json:"-"`
}
type Component struct {
	ID        string    `json:"id"`
	Installed string    `json:"installed"`
	Build     string    `json:"build"`
	Previous  string    `json:"previous"`
	Releases  []Release `json:"releases"`
	Error     string    `json:"error"`
	Source    string    `json:"source"`
}
type Result struct {
	Version   string `json:"version"`
	Installer string `json:"installer,omitempty"`
}
type generation struct {
	Disabled bool              `json:"disabled,omitempty"`
	Version  string            `json:"version"`
	Slot     string            `json:"slot"`
	Hashes   map[string]string `json:"hashes"`
}
type state struct {
	Current     *generation `json:"current,omitempty"`
	Previous    *generation `json:"previous,omitempty"`
	Older       *generation `json:"older,omitempty"`
	AppRollback bool        `json:"appRollback,omitempty"`
}
type asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}
type githubRelease struct {
	Tag        string  `json:"tag_name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Published  string  `json:"published_at"`
	Assets     []asset `json:"assets"`
}
type source struct {
	repo  string
	asset string
	files []string
}

var sources = map[string]source{
	"amneziawg": {"sanderstripa/connect", "Connect-AmneziaWG-windows-amd64.zip", []string{"awg-client.exe"}},
	"app":       {"sanderstripa/connect", "Connect-Setup.exe", nil},
	"xray":      {"XTLS/Xray-core", "Xray-windows-64.zip", []string{"xray.exe"}},
	"sudoku":    {"SUDOKU-ASCII/sudoku", "sudoku-windows-amd64.zip", []string{"sudoku.exe"}},
	// Upstream Mist binaries do not implement Connect's SOCKS adapter.
	"mist": {"sanderstripa/connect", "Connect-Mist-windows-amd64.zip", []string{"mist-client.exe"}},
	"geo":  {"Loyalsoldier/v2ray-rules-dat", "", []string{"geoip.dat", "geosite.dat"}},
}

func requiredFiles(id string) []string {
	if id == "app" {
		return []string{"Connect-Setup.exe"}
	}
	return sources[id].files
}

type Manager struct {
	Root      string
	Baselines map[string]Baseline
	client    *http.Client
	api       string
}

func New(root string, baselines map[string]Baseline) *Manager {
	m := &Manager{Root: root, Baselines: baselines, api: "https://api.github.com", client: &http.Client{Timeout: 5 * time.Minute}}
	m.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 {
			return errors.New("too many redirects")
		}
		return m.allowed(req.URL.String())
	}
	return m
}
func (m *Manager) allowed(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil {
		return errors.New("HTTPS download required")
	}
	api, _ := url.Parse(m.api)
	switch u.Hostname() {
	case "github.com", "api.github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return nil
	}
	if api != nil && u.Host == api.Host {
		return nil
	}
	return errors.New("untrusted update host")
}
func (m *Manager) get(ctx context.Context, raw string, limit int64) ([]byte, error) {
	if e := m.allowed(raw); e != nil {
		return nil, e
	}
	req, e := http.NewRequestWithContext(ctx, "GET", raw, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "Connect/"+AppVersion)
	req.Header.Set("Accept", "application/vnd.github+json")
	res, e := m.client.Do(req)
	if e != nil {
		return nil, fmt.Errorf("download unavailable: %w", e)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("update server HTTP %d", res.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(b)) > limit {
		return nil, errors.New("update exceeds size limit")
	}
	return b, nil
}
func (m *Manager) catalog(ctx context.Context, id string, preview bool) ([]Release, error) {
	s, ok := sources[id]
	if !ok {
		return nil, errors.New("unknown component")
	}
	data, e := m.get(ctx, m.api+"/repos/"+s.repo+"/releases?per_page=30", 8<<20)
	if e != nil {
		return nil, e
	}
	var upstream []githubRelease
	if e = json.Unmarshal(data, &upstream); e != nil {
		return nil, errors.New("invalid release catalog")
	}
	out := []Release{}
	for _, r := range upstream {
		if r.Draft || (!preview && r.Prerelease) {
			continue
		}
		valid := true
		required := []string{s.asset}
		if id == "geo" {
			required = s.files
		}
		for _, name := range required {
			found := false
			for _, a := range r.Assets {
				if a.Name == name && validDigest(a.Digest) {
					found = true
				}
			}
			if !found {
				valid = false
			}
		}
		if valid {
			out = append(out, Release{r.Tag, r.Prerelease, r.Published, r.Assets})
		}
		if len(out) == 10 {
			break
		}
	}
	return out, nil
}
func validDigest(v string) bool {
	if !strings.HasPrefix(v, "sha256:") {
		return false
	}
	b, e := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return e == nil && len(b) == 32
}
func hash(b []byte) string                    { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func (m *Manager) statePath(id string) string { return filepath.Join(m.Root, id, "active.json") }
func (m *Manager) readState(id string) (state, error) {
	if _, ok := sources[id]; !ok {
		return state{}, errors.New("unknown component")
	}
	b, e := os.ReadFile(m.statePath(id))
	if os.IsNotExist(e) {
		return state{}, nil
	}
	if e != nil {
		return state{}, e
	}
	var s state
	e = json.Unmarshal(b, &s)
	return s, e
}
func (m *Manager) slot(id string, g *generation) (string, error) {
	if g == nil || len(g.Slot) != 64 {
		return "", errors.New("invalid component slot")
	}
	if _, e := hex.DecodeString(g.Slot); e != nil {
		return "", e
	}
	return filepath.Join(m.Root, id, g.Slot), nil
}
func (m *Manager) validate(id string, g *generation) error {
	if g != nil && g.Disabled {
		if id == "geo" && g.Slot == "" && len(g.Hashes) == 0 {
			return nil
		}
		return errors.New("invalid disabled slot")
	}
	dir, e := m.slot(id, g)
	if e != nil {
		return e
	}
	if len(g.Hashes) != len(requiredFiles(id)) {
		return errors.New("incomplete component slot")
	}
	for _, name := range requiredFiles(id) {
		expected, ok := g.Hashes[name]
		if !ok {
			return errors.New("missing component file")
		}
		b, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil || hash(b) != expected {
			return errors.New("installed component integrity check failed")
		}
	}
	return nil
}
func (m *Manager) Resolve(id, name, fallback string) (string, error) {
	s, e := m.readState(id)
	if e != nil {
		return "", e
	}
	if s.Current == nil {
		return fallback, nil
	}
	if e = m.validate(id, s.Current); e != nil {
		return "", e
	}
	if s.Current.Disabled {
		return fallback, nil
	}
	if _, ok := s.Current.Hashes[name]; !ok {
		return "", errors.New("unknown component file")
	}
	dir, _ := m.slot(id, s.Current)
	return filepath.Join(dir, name), nil
}
func (m *Manager) Status(id string, check, preview bool) Component {
	c := Component{ID: id, Releases: []Release{}}
	src, ok := sources[id]
	if !ok {
		c.Error = "unknown component"
		return c
	}
	c.Source = src.repo
	c.Installed = m.Baselines[id].Version
	c.Build = m.Baselines[id].Build
	s, e := m.readState(id)
	if e != nil {
		c.Error = e.Error()
	} else {
		if s.Current != nil {
			c.Installed = s.Current.Version
			c.Build = ""
			if s.Current.Disabled {
				c.Installed = ""
			}
			if e = m.validate(id, s.Current); e != nil {
				c.Error = e.Error()
			}
		}
		if s.Previous != nil {
			c.Previous = s.Previous.Version
		}
		if id == "app" {
			c.Installed = m.Baselines[id].Version
			c.Build = ""
			actual := strings.TrimPrefix(c.Installed, "v")
			if s.Current != nil && strings.TrimPrefix(s.Current.Version, "v") != actual && s.Previous != nil && strings.TrimPrefix(s.Previous.Version, "v") == actual {
				candidate := s.Older
				if s.AppRollback {
					candidate = s.Current
				}
				c.Previous = ""
				if candidate != nil {
					c.Previous = candidate.Version
				}
			}
			if strings.TrimPrefix(c.Previous, "v") == strings.TrimPrefix(c.Installed, "v") {
				c.Previous = ""
			}
		}
	}
	if check {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		r, err := m.catalog(ctx, id, preview)
		if err != nil {
			c.Error = err.Error()
		} else {
			c.Releases = r
			if len(r) == 0 {
				c.Error = "No compatible verified packages published"
			}
		}
	}
	return c
}
func (m *Manager) download(ctx context.Context, a asset) ([]byte, error) {
	if !validDigest(a.Digest) {
		return nil, errors.New("SHA-256 digest unavailable")
	}
	if a.Size <= 0 || a.Size > 250<<20 {
		return nil, errors.New("invalid update size")
	}
	b, e := m.get(ctx, a.URL, 250<<20)
	if e != nil {
		return nil, e
	}
	if int64(len(b)) != a.Size || hash(b) != strings.TrimPrefix(a.Digest, "sha256:") {
		return nil, errors.New("download checksum mismatch")
	}
	return b, nil
}
func extract(data []byte, names []string) (map[string][]byte, error) {
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return nil, e
	}
	out := map[string][]byte{}
	for _, f := range z.File {
		for _, name := range names {
			if strings.EqualFold(filepath.Base(strings.ReplaceAll(f.Name, "\\", "/")), name) {
				if _, ok := out[name]; ok {
					return nil, errors.New("duplicate executable in package")
				}
				if f.UncompressedSize64 > 180<<20 {
					return nil, errors.New("oversized archive entry")
				}
				r, e := f.Open()
				if e != nil {
					return nil, e
				}
				b, e := io.ReadAll(io.LimitReader(r, 180<<20+1))
				r.Close()
				if e != nil || len(b) > 180<<20 {
					return nil, errors.New("invalid archive entry")
				}
				out[name] = b
			}
		}
	}
	if len(out) != len(names) {
		return nil, errors.New("component missing from archive")
	}
	return out, nil
}
func verifyExecutable(b []byte) error {
	if len(b) < 64 || string(b[:2]) != "MZ" {
		return errors.New("invalid Windows executable")
	}
	p, e := pe.NewFile(bytes.NewReader(b))
	if e != nil {
		return errors.New("invalid Windows executable")
	}
	defer p.Close()
	if p.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || p.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 {
		return errors.New("Windows x64 executable required")
	}
	return nil
}
func writeAtomic(file string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(file), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(file), ".stage-")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	return replaceFile(name, file)
}
func (m *Manager) save(id string, s state) error {
	b, e := json.MarshalIndent(s, "", "  ")
	if e != nil {
		return e
	}
	return writeAtomic(m.statePath(id), b)
}
func (m *Manager) store(id, version string, files map[string][]byte) (*generation, error) {
	hashes := map[string]string{}
	identity := version
	for _, name := range requiredFiles(id) {
		b, ok := files[name]
		if !ok {
			return nil, errors.New("incomplete update")
		}
		if strings.HasSuffix(name, ".exe") {
			if e := verifyExecutable(b); e != nil {
				return nil, e
			}
		} else if len(b) < 32 {
			return nil, errors.New("empty Geo database")
		}
		hashes[name] = hash(b)
		identity += name + hashes[name]
	}
	g := &generation{Version: version, Slot: hash([]byte(identity)), Hashes: hashes}
	dir, _ := m.slot(id, g)
	if m.validate(id, g) == nil {
		return g, nil
	}
	for name, b := range files {
		if e := writeAtomic(filepath.Join(dir, name), b); e != nil {
			return nil, e
		}
	}
	return g, m.validate(id, g)
}
func (m *Manager) baseline(id string) (*generation, error) {
	b, ok := m.Baselines[id]
	if !ok || len(b.Files) != len(requiredFiles(id)) {
		return nil, nil
	}
	files := map[string][]byte{}
	for name, path := range b.Files {
		data, e := os.ReadFile(path)
		if os.IsNotExist(e) {
			if id == "geo" {
				return &generation{Disabled: true, Version: "Not installed"}, nil
			}
			return nil, nil
		}
		if e != nil {
			return nil, e
		}
		files[name] = data
	}
	if id == "app" && b.Executable != "" {
		exe, e := os.ReadFile(b.Executable)
		if e != nil {
			return nil, e
		}
		if !bytes.Contains(files["Connect-Setup.exe"], exe) {
			return nil, nil
		}
	}
	return m.store(id, b.Version, files)
}
func (m *Manager) Install(id, version string) (Result, error) {
	operations.Lock()
	defer operations.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	releases, e := m.catalog(ctx, id, true)
	if e != nil {
		return Result{}, e
	}
	var selected *Release
	for i := range releases {
		if releases[i].Version == version {
			selected = &releases[i]
			break
		}
	}
	if selected == nil {
		return Result{}, errors.New("release no longer available; check again")
	}
	src := sources[id]
	files := map[string][]byte{}
	for _, a := range selected.Assets {
		wanted := a.Name == src.asset
		if id == "geo" {
			wanted = a.Name == "geoip.dat" || a.Name == "geosite.dat"
		}
		if !wanted {
			continue
		}
		data, e := m.download(ctx, a)
		if e != nil {
			return Result{}, e
		}
		if id == "app" {
			files["Connect-Setup.exe"] = data
			continue
		}
		if id == "geo" {
			files[a.Name] = data
		} else {
			files, e = extract(data, src.files)
			if e != nil {
				return Result{}, e
			}
		}
	}
	g, e := m.store(id, version, files)
	if e != nil {
		return Result{}, e
	}
	s, e := m.readState(id)
	if e != nil {
		return Result{}, e
	}
	if s.Current != nil && s.Current.Slot == g.Slot {
		if id != "app" {
			return Result{Version: version}, nil
		}
	}
	previous := s.Current
	var older *generation
	if id == "app" {
		older = s.Previous
		if previous != nil && strings.TrimPrefix(previous.Version, "v") != strings.TrimPrefix(m.Baselines[id].Version, "v") {
			older = s.Older
		}
	}
	if id == "app" && previous != nil && strings.TrimPrefix(previous.Version, "v") != strings.TrimPrefix(m.Baselines[id].Version, "v") {
		previous = nil
	}
	if previous == nil {
		previous, e = m.baseline(id)
		if e != nil {
			return Result{}, e
		}
	}
	if e = m.save(id, state{Current: g, Previous: previous, Older: older}); e != nil {
		return Result{}, e
	}
	result := Result{Version: version}
	if id == "app" {
		dir, _ := m.slot(id, g)
		result.Installer = filepath.Join(dir, "Connect-Setup.exe")
	}
	return result, nil
}
func (m *Manager) Rollback(id string) (Result, error) {
	operations.Lock()
	defer operations.Unlock()
	s, e := m.readState(id)
	if e != nil {
		return Result{}, e
	}
	if id == "app" && s.Current != nil && s.Previous != nil && strings.TrimPrefix(s.Current.Version, "v") != strings.TrimPrefix(m.Baselines[id].Version, "v") && strings.TrimPrefix(s.Previous.Version, "v") == strings.TrimPrefix(m.Baselines[id].Version, "v") {
		if s.AppRollback {
			s.Current, s.Previous = s.Previous, s.Current
		} else {
			s.Current = s.Previous
			s.Previous = s.Older
		}
	}
	if s.Previous == nil {
		return Result{}, errors.New("no previous installed version")
	}
	if e = m.validate(id, s.Previous); e != nil {
		return Result{}, e
	}
	s.Current, s.Previous = s.Previous, s.Current
	if id == "app" {
		s.AppRollback = true
	}
	if e = m.save(id, s); e != nil {
		return Result{}, e
	}
	result := Result{Version: s.Current.Version}
	if id == "app" {
		dir, _ := m.slot(id, s.Current)
		result.Installer = filepath.Join(dir, "Connect-Setup.exe")
	}
	return result, nil
}
