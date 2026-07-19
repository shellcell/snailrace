package model

import (
	"encoding/json"
	"testing"
)

func TestLegacyRunInfersPhysicalFootprintValidity(t *testing.T) {
	var run Run
	if err := json.Unmarshal(
		[]byte(`{"peak_physical_footprint_bytes":123}`), &run,
	); err != nil {
		t.Fatal(err)
	}
	if !run.PhysicalFootprintValid {
		t.Fatal("legacy positive physical footprint should remain valid")
	}
}

func TestLegacyToolInfersShellCommand(t *testing.T) {
	var tool ToolInfo
	if err := json.Unmarshal([]byte(`{"command":["grep foo file"]}`), &tool); err != nil {
		t.Fatal(err)
	}
	if !tool.ShellCommand {
		t.Fatal("legacy one-element command should retain shell rendering")
	}
	data, err := json.Marshal(ToolInfo{Command: []string{"/tmp/my tool"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &tool); err != nil {
		t.Fatal(err)
	}
	if tool.ShellCommand {
		t.Fatal("current direct command should retain argv rendering")
	}
}

func TestLegacyToolInfersDirectSingleArgumentAndVerifiedHash(t *testing.T) {
	var tool ToolInfo
	if err := json.Unmarshal([]byte(
		`{"command":["/tmp/my tool"],"executable":"/tmp/my tool","sha256":"abc"}`,
	), &tool); err != nil {
		t.Fatal(err)
	}
	if tool.ShellCommand || !tool.ProvenanceVerified {
		t.Fatalf("legacy direct tool = %+v", tool)
	}
}
