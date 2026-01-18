package domain

import (
	"testing"
	"time"
)

func TestGenerateETag_Deterministic(t *testing.T) {
	// Same ICS → same ETag
	ics := []byte("BEGIN:VCALENDAR\nVERSION:2.0\nEND:VCALENDAR")

	etag1 := GenerateETag(ics)
	etag2 := GenerateETag(ics)

	if etag1 != etag2 {
		t.Errorf("ETags should be equal for same ICS: %s != %s", etag1, etag2)
	}

	// ETag should be quoted
	if etag1[0] != '"' || etag1[len(etag1)-1] != '"' {
		t.Errorf("ETag should be quoted: %s", etag1)
	}
}

func TestGenerateETag_DifferentICS(t *testing.T) {
	// Different ICS → different ETag
	ics1 := []byte("BEGIN:VCALENDAR\nVERSION:2.0\nEND:VCALENDAR")
	ics2 := []byte("BEGIN:VCALENDAR\nVERSION:2.1\nEND:VCALENDAR")

	if GenerateETag(ics1) == GenerateETag(ics2) {
		t.Error("ETags should differ for different ICS")
	}
}

func TestEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		wantErr bool
	}{
		{
			name: "valid event",
			event: Event{
				UID:       "test-uid",
				ICS:       "BEGIN:VEVENT...",
				StartTime: time.Now(),
				EndTime:   time.Now().Add(time.Hour),
			},
			wantErr: false,
		},
		{
			name: "missing UID",
			event: Event{
				ICS:       "BEGIN:VEVENT...",
				StartTime: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing ICS",
			event: Event{
				UID:       "test-uid",
				StartTime: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "end before start",
			event: Event{
				UID:       "test-uid",
				ICS:       "BEGIN:VEVENT...",
				StartTime: time.Now(),
				EndTime:   time.Now().Add(-time.Hour),
			},
			wantErr: true,
		},
		{
			name: "missing start time",
			event: Event{
				UID: "test-uid",
				ICS: "BEGIN:VEVENT...",
			},
			wantErr: true,
		},
		{
			name: "valid event with zero end time",
			event: Event{
				UID:       "test-uid",
				ICS:       "BEGIN:VEVENT...",
				StartTime: time.Now(),
				EndTime:   time.Time{}, // Zero time is valid (no end time)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
