package gopenqa

import "testing"

func TestJobProgress(t *testing.T) {
	tests := []struct {
		name    string
		modules []JobModule
		wantPct int
		wantOK  bool
	}{
		{"no modules", nil, 0, false},
		{"all pending", []JobModule{{Result: "none"}, {Result: "none"}}, 0, true},
		{"mixed results", []JobModule{{Result: "passed"}, {Result: "softfailed"}, {Result: "failed"}, {Result: "none"}}, 75, true},
		{"running module counts towards total", []JobModule{{Result: "passed"}, {Result: "running"}, {Result: "none"}}, 33, true},
		{"all done", []JobModule{{Result: "passed"}, {Result: "softfailed"}}, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := Job{Modules: tt.modules}
			pct, ok := job.Progress()
			if ok != tt.wantOK {
				t.Fatalf("Progress() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && pct != tt.wantPct {
				t.Fatalf("Progress() = %d%%, want %d%%", pct, tt.wantPct)
			}
		})
	}
}
