package main

import (
	"github.com/SUDOKU-ASCII/sudoku-desktop/internal/updates"
	"github.com/wailsapp/wails/v3/pkg/application"
	"time"
)

func (a *App) GetUpdateStatus(id string, check, preview bool) (updates.Component, error) {
	if a.remote != nil {
		var result updates.Component
		e := a.remote.call("GetUpdateStatus", []any{id, check, preview}, &result)
		return result, e
	}
	return a.backend.GetUpdateStatus(id, check, preview)
}
func (a *App) UpdateComponent(id, version string) (updates.Result, error) {
	var result updates.Result
	var err error
	if a.remote != nil {
		err = a.remote.call("UpdateComponent", []any{id, version}, &result)
	} else {
		result, err = a.backend.UpdateComponent(id, version)
	}
	return a.finishUpdate(result, err)
}
func (a *App) finishUpdate(result updates.Result, err error) (updates.Result, error) {
	if err != nil {
		return result, err
	}
	// The network helper stages files only. The UI owns the installer handoff.
	if result.Installer != "" && application.Get() != nil {
		if err = a.StopProxy(); err != nil {
			return result, err
		}
		if err = launchUpdateInstaller(result.Installer); err != nil {
			return result, err
		}
		go func() {
			time.Sleep(500 * time.Millisecond)
			if app := application.Get(); app != nil {
				app.Event.Emit("tray:quit")
			}
		}()
	}
	return result, nil
}
func (a *App) RollbackComponent(id string) (updates.Result, error) {
	var result updates.Result
	var e error
	if a.remote != nil {
		e = a.remote.call("RollbackComponent", []any{id}, &result)
	} else {
		result, e = a.backend.RollbackComponent(id)
	}
	return a.finishUpdate(result, e)
}
