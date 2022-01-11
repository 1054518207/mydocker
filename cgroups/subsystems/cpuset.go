package subsystems

import (
	"fmt"
	"os"
	"path"
)

const (
	cpusetCpus = "cpuset.cpus"
)

type CpusetSubSystem struct {
}

func (s *CpusetSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath); err == nil {
		if res.CpuSet != "" {
			err := os.WriteFile(path.Join(subsysCgroupPath, cpusetCpus), []byte(res.CpuSet), 0644)
			if err != nil {
				return fmt.Errorf("set cgroup cpuset fail %v", err)
			}
		}
		return nil
	} else {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
}

func (s *CpusetSubSystem) Remove(cgroupPath string) error {
	return remove(s.Name(), cgroupPath)
}

func (s *CpusetSubSystem) Apply(cgroupPath string, pid int) error {
	return apply(s.Name(), cgroupPath, pid)
}

func (s *CpusetSubSystem) Name() string {
	return "cpuset"
}
