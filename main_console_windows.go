//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

const attachParentProcess = ^uintptr(0)

var (
	kernel32DLL       = windows.NewLazySystemDLL("kernel32.dll")
	attachConsoleProc = kernel32DLL.NewProc("AttachConsole")
)

func prepareConsoleForCLI(args []string) {
	if !shouldAttachConsoleForCLI(args) {
		return
	}

	if err := attachToParentConsole(); err != nil {
		return
	}

	if err := bindConsoleStreams(); err != nil {
		return
	}

	_ = ensureConsoleStdHandles()
}

func attachToParentConsole() error {
	if err := attachConsoleProc.Find(); err != nil {
		return err
	}

	r1, _, callErr := attachConsoleProc.Call(attachParentProcess)
	if r1 != 0 {
		return nil
	}

	errno, ok := callErr.(syscall.Errno)
	if !ok {
		return callErr
	}

	if errno == 0 || errno == windows.ERROR_ACCESS_DENIED {
		return nil
	}

	return errno
}

func bindConsoleStreams() error {
	stdin, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return err
	}

	stdout, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		_ = stdin.Close()
		return err
	}

	stderr, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return err
	}

	os.Stdin = stdin
	os.Stdout = stdout
	os.Stderr = stderr

	_ = windows.SetStdHandle(windows.STD_INPUT_HANDLE, windows.Handle(stdin.Fd()))
	_ = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(stdout.Fd()))
	_ = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(stderr.Fd()))

	return nil
}

func ensureConsoleStdHandles() error {
	for _, stdHandle := range []uint32{
		windows.STD_INPUT_HANDLE,
		windows.STD_OUTPUT_HANDLE,
		windows.STD_ERROR_HANDLE,
	} {
		handle, err := windows.GetStdHandle(stdHandle)
		if err != nil {
			return err
		}
		if handle == 0 || handle == windows.InvalidHandle {
			return fmt.Errorf("std handle %d is invalid", stdHandle)
		}
	}
	return nil
}
