//go:build darwin

package collector

import "syscall"

type osFileSystems struct{}

func (osFileSystems) StatFS(path string) (FileSystemStats, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return FileSystemStats{}, &UnavailableError{Metric: "statfs", Target: path, Reason: err.Error()}
	}
	return FileSystemStats{
		Blocks: stats.Blocks,
		Bfree:  stats.Bfree,
		Files:  stats.Files,
		Ffree:  stats.Ffree,
	}, nil
}
