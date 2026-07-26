package logger

import "testing"

func TestNewMCPLoggerRedirectsStdout(t *testing.T) {
	testCases := []struct {
		name   string
		output string
		want   string
	}{
		{name: "stdout", output: "stdout", want: "stderr"},
		{name: "case and whitespace", output: " STDOUT ", want: "stderr"},
		{name: "empty", output: "", want: "stderr"},
		{name: "stderr", output: "stderr", want: "stderr"},
		{name: "file", output: "/tmp/sshmcp.log", want: "/tmp/sshmcp.log"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mcpSafeOutput(tc.output); got != tc.want {
				t.Fatalf("MCP logger output = %q, want %q", got, tc.want)
			}
		})
	}
}
