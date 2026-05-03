//go:build !linux

package collector

type osFileSystems struct{}

func (osFileSystems) StatFS(path string) (FileSystemStats, error) {
	return FileSystemStats{}, &UnavailableError{Metric: "statfs", Target: path, Reason: "Linux collectors are not supported on this operating system"}
}
