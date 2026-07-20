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
	"unsafe"
)

// GroupSampler owns the scratch buffers for one monitoring goroutine, so the
// per-tick sampling loop does not generate garbage while a run is measured.
// It is not safe for concurrent use.
type GroupSampler struct {
	pids []C.pid_t
}

func NewGroupSampler() *GroupSampler { return &GroupSampler{} }

func SampleProcessGroup(rootPID int) (Metrics, bool) {
	return NewGroupSampler().Sample(rootPID)
}

func (sampler *GroupSampler) Sample(rootPID int) (Metrics, bool) {
	pids := sampler.processGroupPIDs(rootPID)
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
			if syscall.Kill(int(pid), 0) == nil {
				physicalFootprintValid = false
			}
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

func (sampler *GroupSampler) processGroupPIDs(groupID int) []C.pid_t {
	bytes := C.proc_listpgrppids(C.pid_t(groupID), nil, 0)
	if bytes <= 0 {
		return nil
	}
	// Leave room for process churn, and retry if the result fills the buffer.
	capacity := int(bytes) + 16
	for range 3 {
		if cap(sampler.pids) < capacity {
			sampler.pids = make([]C.pid_t, capacity)
		}
		pids := sampler.pids[:capacity]
		count := int(C.proc_listpgrppids(
			C.pid_t(groupID), unsafe.Pointer(&pids[0]), C.int(len(pids))*C.sizeof_pid_t,
		))
		if count <= 0 {
			return nil
		}
		if count < len(pids) {
			return pids[:count]
		}
		capacity *= 2
	}
	return nil
}
