package jvm

import (
	"bytes"
	"fmt"
)

func DumpPerfmap(pid uint32) error {
	j, err := Dial(pid)
	if err != nil {
		return fmt.Errorf("failed to attach to JVM %d: %w", pid, err)
	}
	defer j.Close()
	if err = j.DumpPerfmap(); err != nil {
		return fmt.Errorf("failed to dump perfmap of JVM %d: %w", pid, err)
	}
	return nil
}

func GetVMFlags(pid uint32) (string, error) {
	j, err := Dial(pid)
	if err != nil {
		return "", fmt.Errorf("failed to attach to JVM %d: %w", pid, err)
	}
	defer j.Close()
	vmFlags, err := j.GetVMFlags()
	if err != nil {
		return "", fmt.Errorf("failed to get VM flags of JVM %d: %w", pid, err)
	}
	return vmFlags, nil
}

func GetSystemProperties(pid uint32) (string, error) {
	j, err := Dial(pid)
	if err != nil {
		return "", fmt.Errorf("failed to attach to JVM %d: %w", pid, err)
	}
	defer j.Close()
	props, err := j.GetSystemProperties()
	if err != nil {
		return "", fmt.Errorf("failed to get system properties of JVM %d: %w", pid, err)
	}
	return props, nil
}

func GetVersion(pid uint32) (string, error) {
	j, err := Dial(pid)
	if err != nil {
		return "", fmt.Errorf("failed to attach to JVM %d: %w", pid, err)
	}
	defer j.Close()
	version, err := j.GetVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get version of JVM %d: %w", pid, err)
	}
	return version, nil
}

func IsPerfmapDumpSupported(cmdline []byte) bool {
	if !bytes.Contains(cmdline, []byte("-XX:+PreserveFramePointer")) {
		return false
	}
	return true
}
