//go:build !linux && !darwin

package collector

type osFileSystems struct{}

func (osFileSystems) StatFS(path string) (FileSystemStats, error) {
	return FileSystemStats{}, &UnavailableError{Metric: "statfs", Target: path, Reason: "collectors are not supported on this operating system"}
}
