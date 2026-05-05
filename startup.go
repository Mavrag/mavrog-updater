package main

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const startupRegKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const startupRegName = "MavrogUpdater"

func setRunOnStartup(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, startupRegKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if enable {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		exe, _ = filepath.Abs(exe)
		return k.SetStringValue(startupRegName, `"`+exe+`"`)
	}
	return k.DeleteValue(startupRegName)
}

func getRunOnStartup() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, startupRegKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(startupRegName)
	return err == nil
}
