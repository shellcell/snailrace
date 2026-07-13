package runner

import "testing"

func TestInspectToolReportsDyldSharedCacheDependencies(t *testing.T) {
	tool, err := inspectTool(Spec{Name: "sleep", Args: []string{"/bin/sleep", "0"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tool.SharedCacheFiles) == 0 {
		t.Fatal("expected /bin/sleep to report a dyld shared-cache dependency")
	}
	if tool.DiskFootprintBytes < tool.SizeBytes {
		t.Fatalf("disk footprint %d is smaller than executable %d", tool.DiskFootprintBytes, tool.SizeBytes)
	}
}
