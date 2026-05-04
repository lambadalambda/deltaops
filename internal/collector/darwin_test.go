package collector

import (
	"context"
	"testing"
)

func TestDarwinCollectorCollectsFilesystemSamplesOnly(t *testing.T) {
	collector, err := NewCollector("darwin", Dependencies{
		FileSystems: fakeFileSystems{
			"/": FileSystemStats{Blocks: 200, Bfree: 50, Files: 20, Ffree: 5},
		},
		Proc: fakeProc{},
	}, []string{"/"})
	if err != nil {
		t.Fatalf("NewCollector returned error: %v", err)
	}

	samples, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(samples) != 2 {
		t.Fatalf("sample count = %d, want filesystem-only samples: %#v", len(samples), samples)
	}
	assertSample(t, samples, MetricDiskUsedPercent, "/", 75)
	assertSample(t, samples, MetricDiskInodesUsedPercent, "/", 75)
}
