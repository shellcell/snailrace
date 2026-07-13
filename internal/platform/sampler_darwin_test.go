package platform

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestSampleTreeReadsProcessGroup(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	metrics, valid := SampleTree(cmd.Process.Pid)
	if !valid {
		t.Fatal("process group leader was not sampled")
	}
	if metrics.ResidentBytes == 0 || metrics.PhysicalFootprintBytes == 0 {
		t.Fatalf("memory metrics were not sampled: %+v", metrics)
	}
	if metrics.Processes != 1 || metrics.Threads == 0 {
		t.Fatalf("invalid process totals: %+v", metrics)
	}
}
