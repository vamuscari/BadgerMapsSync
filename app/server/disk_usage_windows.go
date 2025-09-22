//go:build windows

package server

import "golang.org/x/sys/windows"

func diskUsage(path string) (available uint64, total uint64, err error) {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, err
	}

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	if err = windows.GetDiskFreeSpaceEx(ptr, &freeBytesAvailable, &totalNumberOfBytes, &totalNumberOfFreeBytes); err != nil {
		return 0, 0, err
	}

	return freeBytesAvailable, totalNumberOfBytes, nil
}
