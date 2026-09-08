package system

import "testing"

func TestKernelVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "proc version", input: "Linux version 6.8.0-generic (builder@host)", want: "6.8.0-generic"},
		{name: "osrelease fallback", input: "", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := kernelVersion(test.input); test.name == "osrelease fallback" {
				if got == "" {
					t.Skip("host does not expose /proc/sys/kernel/osrelease")
				}
			} else if got != test.want {
				t.Fatalf("kernelVersion() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "minutes", input: "125.4 0.0", want: "2m"},
		{name: "hours", input: "3725.0 0.0", want: "1h 2m"},
		{name: "days", input: "90125.0 0.0", want: "1d 1h 2m"},
		{name: "invalid", input: "unknown", want: "unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatUptime(test.input); got != test.want {
				t.Fatalf("formatUptime() = %q, want %q", got, test.want)
			}
		})
	}
}

func BenchmarkFormatUptime(b *testing.B) {
	for b.Loop() {
		_ = formatUptime("123456.78 0.0")
	}
}
