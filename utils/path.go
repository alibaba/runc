package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindAbPathInRootfs(path string, rootfs string, sys_paths []string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	for _, s_path := range sys_paths {
		real_path := filepath.Join(rootfs, s_path)
		_, err := os.Stat(filepath.Join(real_path, path))

		if err == nil {
			return filepath.Join(s_path, path), nil
		}

		if os.IsNotExist(err) {
			continue
		}

		return "", err
	}

	return "", fmt.Errorf("not found %s in PATH", path)
}
