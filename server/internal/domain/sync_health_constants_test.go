package domain

import "testing"

func TestSyncHealthStatusStringValues(t *testing.T) {
	cases := map[string]SyncHealthStatus{
		"healthy":  SyncHealthHealthy,
		"warning":  SyncHealthWarning,
		"critical": SyncHealthCritical,
		"unknown":  SyncHealthUnknown,
	}

	for want, status := range cases {
		if string(status) != want {
			t.Fatalf("status string mismatch: got %q, want %q", status, want)
		}
	}
}
