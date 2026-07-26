package serialmcp

import "testing"

func TestBuildModeNormalizesDefaults(t *testing.T) {
	mode, normalized, err := buildMode(Config{Device: "/dev/ttyTEST"})
	if err != nil {
		t.Fatalf("build mode: %v", err)
	}
	if mode.BaudRate != 115200 || normalized.DataBits != 8 || normalized.Parity != "none" || normalized.StopBits != "1" {
		t.Fatalf("unexpected normalized config: mode=%+v config=%+v", mode, normalized)
	}
}

func TestBuildModeRejectsInvalidSettings(t *testing.T) {
	cases := []Config{
		{Device: "/dev/ttyTEST", BaudRate: -1},
		{Device: "/dev/ttyTEST", DataBits: 9},
		{Device: "/dev/ttyTEST", Parity: "invalid"},
		{Device: "/dev/ttyTEST", StopBits: "3"},
	}
	for _, config := range cases {
		if _, _, err := buildMode(config); err == nil {
			t.Fatalf("expected invalid config to fail: %+v", config)
		}
	}
}
