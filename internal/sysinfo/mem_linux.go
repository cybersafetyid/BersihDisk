//go:build linux

package sysinfo

import (
	"bytes"
	"os"
	"strconv"
)

func totalMemory() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		if !bytes.HasPrefix(line, []byte("MemTotal:")) {
			continue
		}
		fields := bytes.Fields(line[len("MemTotal:"):])
		if len(fields) == 0 {
			return 0
		}
		kb, err := strconv.ParseUint(string(fields[0]), 10, 64)
		if err != nil {
			return 0
		}
		return kb * 1024
	}
	return 0
}
