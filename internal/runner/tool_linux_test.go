package runner

import "testing"

func TestInspectToolIncludesLinkedFootprint(t *testing.T) {
	tool, err := inspectTool(Spec{Name: "sleep", Args: []string{"/bin/sleep", "0"}})
	if err != nil {
		t.Fatal(err)
	}
	if tool.DiskFootprintBytes < tool.SizeBytes {
		t.Fatalf("footprint %d is smaller than executable %d", tool.DiskFootprintBytes, tool.SizeBytes)
	}
	if tool.LinkedSizeBytes == 0 || len(tool.LinkedFiles) == 0 {
		t.Fatal("expected dynamically linked files for /bin/sleep")
	}
}
