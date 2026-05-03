//go:build linux

package collector

import "syscall"

type osFileSystems struct{}

func (osFileSystems) StatFS(path string) (FileSystemStats, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return FileSystemStats{}, &UnavailableError{Metric: "statfs", Target: path, Reason: err.Error()}
	}
	return FileSystemStats{
		Blocks: uint64(stats.Blocks),
		Bfree:  uint64(stats.Bfree),
		Files:  uint64(stats.Files),
		Ffree:  uint64(stats.Ffree),
	}, nil
}
