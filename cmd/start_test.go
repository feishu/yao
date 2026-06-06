package cmd

import "testing"

func TestStartCmdHasInspectFlag(t *testing.T) {
	flag := startCmd.PersistentFlags().Lookup("inspect")
	if flag == nil {
		t.Fatal("expected start command to register --inspect")
	}
	if flag.DefValue != "" {
		t.Fatalf("expected empty default inspect value, got %q", flag.DefValue)
	}
}
