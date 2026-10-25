package dex

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing/fstest"
)

// loadThemeFS reads a frontend directory into memory and returns it as an
// fs.FS for Dex.
//
// Dex itself would use os.DirFS on the directory and list templates/ to find
// its templates. On Kubernetes a ConfigMap or projected volume is laid out
// through a "..data" symlink and a "..<timestamp>" directory, and a symlink
// is not a directory entry, so Dex would try to parse "..data" as a template
// and refuse to start. Snapshotting the tree here, following symlinks and
// keeping only regular files whose names do not start with "..", makes any
// mount layout work. The snapshot is taken once at start-up: a changed theme
// needs a restart, which is also how the built-in pages behave.
func loadThemeFS(dir string) (fs.FS, error) {
	root := filepath.Clean(dir)
	m := fstest.MapFS{}
	var walk func(abs, rel string) error
	walk = func(abs, rel string) error {
		entries, err := os.ReadDir(abs)
		if err != nil {
			return err
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "..") {
				continue
			}
			childAbs := filepath.Join(abs, name)
			childRel := path.Join(rel, name)
			info, err := os.Stat(childAbs) // follows symlinks
			if err != nil {
				return err
			}
			if info.IsDir() {
				if err := walk(childAbs, childRel); err != nil {
					return err
				}
				continue
			}
			if !info.Mode().IsRegular() {
				continue
			}
			data, err := os.ReadFile(childAbs)
			if err != nil {
				return err
			}
			m[childRel] = &fstest.MapFile{Data: data, Mode: info.Mode().Perm(), ModTime: info.ModTime()}
		}
		return nil
	}
	if err := walk(root, "."); err != nil {
		return nil, fmt.Errorf("load frontend dir %q: %w", root, err)
	}
	if len(m) == 0 {
		return nil, fmt.Errorf("load frontend dir %q: no files", root)
	}
	return m, nil
}
