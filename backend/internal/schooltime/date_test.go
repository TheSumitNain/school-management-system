package schooltime

import (
	"testing"
	"time"
)

func TestDateAtUsesAsiaKolkataCalendarBoundary(t *testing.T) {
	tests := []struct {
		name string
		instant time.Time
		want string
	}{
		{
			name: "before Indian midnight",
			instant: time.Date(2026, time.November, 17, 18, 29, 59, 0, time.UTC),
			want: "2026-11-17",
		},
		{
			name: "at Indian midnight",
			instant: time.Date(2026, time.November, 17, 18, 30, 0, 0, time.UTC),
			want: "2026-11-18",
		},
		{
			name: "after Indian midnight",
			instant: time.Date(2026, time.November, 17, 20, 0, 0, 0, time.UTC),
			want: "2026-11-18",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DateAt(tt.instant); got != tt.want {
				t.Fatalf("DateAt(%s) = %s, want %s", tt.instant.Format(time.RFC3339), got, tt.want)
			}
		})
	}
}

func TestTodayUsesAsiaKolkataLocation(t *testing.T) {
	today := Today()
	if today.Location().String() != TimeZone {
		t.Fatalf("Today() location = %q, want %q", today.Location(), TimeZone)
	}
	if got := today.Format("15:04:05"); got != "00:00:00" {
		t.Fatalf("Today() time = %s, want midnight", got)
	}
}
