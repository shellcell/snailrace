package platform

/*
#include <libproc.h>
#include <sys/proc_info.h>
#include <sys/resource.h>

static int sample_process(pid_t pid, struct proc_taskinfo *task,
                          struct rusage_info_v4 *usage) {
	int size = proc_pidinfo(pid, PROC_PIDTASKINFO, 0, task, sizeof(*task));
	if (size != sizeof(*task)) {
		return 0;
	}
	if (proc_pid_rusage(pid, RUSAGE_INFO_V4, (rusage_info_t *)usage) != 0) {
		usage->ri_phys_footprint = 0;
		return 1;
	}
	return 2;
}
*/
import "C"

import (
	"syscall"
	"time"
	"unsafe"
)

func DefaultInterval() time.Duration { return 10 * time.Millisecond }

func SampleImmediately() bool { return true }

func SampleTree(rootPID int) (Metrics, bool) {
	pids := processGroupPIDs(rootPID)
	var total Metrics
	foundRoot := false
	physicalFootprintValid := true
	for _, pid := range pids {
		if pid <= 0 {
			continue
		}
		var task C.struct_proc_taskinfo
		var usage C.struct_rusage_info_v4
		sampled := C.sample_process(pid, &task, &usage)
		if sampled == 0 {
			physicalFootprintValid = false
			continue
		}
		if sampled < 2 {
			physicalFootprintValid = false
		}
		if int(pid) == rootPID {
			foundRoot = true
		}
		total.ResidentBytes += uint64(task.pti_resident_size)
		total.PhysicalFootprintBytes += uint64(usage.ri_phys_footprint)
		total.VirtualBytes += uint64(task.pti_virtual_size)
		total.Processes++
		total.Threads += uint64(task.pti_threadnum)
	}
	total.PhysicalFootprintValid = total.Processes > 0 && physicalFootprintValid
	total.PhysicalFootprintInvalid = !total.PhysicalFootprintValid &&
		(foundRoot || syscall.Kill(rootPID, 0) == nil)
	return total, foundRoot
}

func processGroupPIDs(groupID int) []C.pid_t {
	bytes := C.proc_listpgrppids(C.pid_t(groupID), nil, 0)
	if bytes <= 0 {
		return nil
	}
	// Leave room for processes created between the sizing and data calls.
	count := int(bytes) + 16
	pids := make([]C.pid_t, count)
	count = int(C.proc_listpgrppids(
		C.pid_t(groupID), unsafe.Pointer(&pids[0]), C.int(len(pids))*C.sizeof_pid_t,
	))
	if count <= 0 {
		return nil
	}
	count = min(count, len(pids))
	return pids[:count]
}
