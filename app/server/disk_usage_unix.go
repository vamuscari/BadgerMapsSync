//go:build !windows

package server

import "syscall"

func diskUsage(path string) (available uint64, total uint64, err error) {
	var stat syscall.Statfs_t
	if err = syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}

	available = stat.Bavail * uint64(stat.Bsize)
	total = stat.Blocks * uint64(stat.Bsize)
	return available, total, nil
}
