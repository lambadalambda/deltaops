package runtime

import (
	"fmt"
	"strings"

	"deltaops/internal/collector"
)

type ReportReason string

const (
	ReportReasonStartup ReportReason = "startup"
	ReportReasonPaired  ReportReason = "paired"
)

type Report struct {
	Reason  ReportReason
	Host    string
	Samples []collector.Sample
}

func (r Report) Message() string {
	var b strings.Builder
	b.WriteString("DeltaOps status report\n")
	if r.Reason != "" {
		fmt.Fprintf(&b, "reason=%s\n", r.Reason)
	}
	if strings.TrimSpace(r.Host) != "" {
		fmt.Fprintf(&b, "host=%s\n", r.Host)
	}
	if len(r.Samples) == 0 {
		b.WriteString("metrics: no samples collected")
		return b.String()
	}
	b.WriteString("metrics:\n")
	for _, sample := range r.Samples {
		fmt.Fprintf(&b, "- check=%s target=%s observed=%.2f\n", sample.Metric, sample.Target, sample.Value)
	}
	return strings.TrimRight(b.String(), "\n")
}
