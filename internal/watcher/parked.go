package watcher

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// Watch monitors the given directories for new and deleted project subdirectories.
// onNew is called when an artisan file appears in a direct subdirectory of a parked dir.
// onRemoved is called when a watched subdirectory is deleted.
func Watch(dirs []string, onNew func(path string), onRemoved func(path string)) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	// parkedDirs tracks the top-level parked directories so we only register
	// projects that are direct children of them, not deeper nestings.
	parkedDirs := map[string]bool{}

	for _, dir := range dirs {
		expanded := expandHome(dir)
		if err := os.MkdirAll(expanded, 0755); err != nil {
			continue
		}
		if err := w.Add(expanded); err != nil {
			continue
		}
		parkedDirs[expanded] = true
		// Also watch existing direct subdirectories so we catch artisan creation inside them.
		entries, _ := os.ReadDir(expanded)
		for _, e := range entries {
			if e.IsDir() {
				sub := filepath.Join(expanded, e.Name())
				if err := w.Add(sub); err != nil {
					logger.Error("failed to watch subdirectory", "path", sub, "err", err)
				}
			}
		}
	}

	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			switch {
			case event.Op&fsnotify.Remove != 0:
				onRemoved(event.Name)
			case event.Op&(fsnotify.Create|fsnotify.Write) != 0:
				if filepath.Base(event.Name) == "artisan" {
					projectDir := filepath.Dir(event.Name)
					// Only register if this is a direct child of a parked dir.
					if parkedDirs[filepath.Dir(projectDir)] {
						onNew(projectDir)
					}
				} else if event.Op&fsnotify.Create != 0 {
					// New direct subdirectory in a parked dir — watch it for artisan.
					if parkedDirs[filepath.Dir(event.Name)] {
						if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
							if err := w.Add(event.Name); err != nil {
								logger.Error("failed to watch new subdirectory", "path", event.Name, "err", err)
							} else {
								logger.Debug("watching new subdirectory", "path", event.Name)
							}
						}
					}
				}
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			logger.Error("fsnotify error", "err", err)
		}
	}
}

func expandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
