package ports

import (
	"fmt"
	"time"
)

// HumanDuration renders a duration compactly for a status table: "3s", "5m",
// "2h", "4d". It is intentionally coarse — single-unit precision is enough to
// answer "how long has this been holding the port?".
func HumanDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// HumanUptime is a convenience wrapper for a listener's uptime.
func (l Listener) HumanUptime() string { return HumanDuration(l.Uptime()) }
