//go:build darwin

package collector

import "testing"

func TestDarwinOSFileSystemsStatFSRoot(t *testing.T) {
	stats, err := osFileSystems{}.StatFS("/")
	if err != nil {
		t.Fatalf("StatFS returned error: %v", err)
	}
	if stats.Blocks == 0 {
		t.Fatalf("Blocks = 0, want nonzero root filesystem capacity")
	}
}
