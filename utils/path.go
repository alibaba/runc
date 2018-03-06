package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindAbPathInRootfs(path string, rootfs string, sysPaths []string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	for _, sPath := range sysPaths {
		realPath := filepath.Join(rootfs, sPath)
		_, err := os.Stat(filepath.Join(realPath, path))

		if err == nil {
			return filepath.Join(sPath, path), nil
		}

		if os.IsNotExist(err) {
			continue
		}

		return "", err
	}

	return "", fmt.Errorf("not found %s in PATH", path)
}
