package platform

import (
	"os/exec"
	"syscall"
	"testing"
)

func BenchmarkSampleProcessGroup(b *testing.B) {
	command := exec.Command("sleep", "60")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		_ = command.Wait()
	})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, valid := SampleProcessGroup(command.Process.Pid); !valid {
			b.Fatal("sampled process disappeared")
		}
	}
}
