package core

import "io/fs"

// Config describes presentation and capabilities, independently of installation code.
type Config struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Path         string `json:"path"`
	Chrome       bool   `json:"chrome"`
	PostInstall  string `json:"postInstall,omitempty"`
	Ribbon       fs.FS  `json:"-"`
	RibbonBounds []int  `json:"ribbonBounds,omitempty"`
}
type Options struct {
	Desktop   bool   `json:"desktop"`
	StartMenu bool   `json:"startMenu"`
	Chrome    bool   `json:"chrome"`
	Path      string `json:"path"`
	Language  string `json:"language"`
	Theme     string `json:"theme"`
}
type Progress struct {
	Percent       int    `json:"progress"`
	Stage         string `json:"stage"`
	Indeterminate bool   `json:"indeterminate"`
}
type Adapter interface {
	Install(Options) error
	Remove() error
	Open() error
	Action(string, []byte) (any, error)
	Progress() Progress
}
