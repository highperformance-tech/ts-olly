//go:build !windows

package fileid

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func queryFilenameById(path string) (uint64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("stat file %s: %w", path, err)
	}
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("os.Fileinfo.Sys() is not syscall.Stat_t")
	}
	return stat.Ino, nil
}
