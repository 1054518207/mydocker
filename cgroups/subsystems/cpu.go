package subsystems

import (
	"fmt"
	"os"
	"path"
)

const (
	cpuShares = "cpu.shares"
)

type CpuSubSystem struct{}

func (s *CpuSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath); err == nil {
		if res.CpuShare != "" {
			err := os.WriteFile(path.Join(subsysCgroupPath, cpuShares), []byte(res.CpuShare), 0644)
			if err != nil {
				return fmt.Errorf("set cgroup cpuset fail %v", err)
			}
		}
		return nil
	} else {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
}

func (s *CpuSubSystem) Remove(cgroupPath string) error {
	return remove(s.Name(), cgroupPath)
}

func (s *CpuSubSystem) Apply(cgroupPath string, pid int) error {
	return apply(s.Name(), cgroupPath, pid)
}

func (s *CpuSubSystem) Name() string {
	return "cpu,cpuacct"
}
