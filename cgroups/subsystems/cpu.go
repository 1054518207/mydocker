package subsystems

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

type CpuSubSystem struct {
}

func (s *CpuSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPat(s.Name(), cgroupPath, true); err == nil {
		if res.CpuShare != "" {
			err := os.WriteFile(path.Join(subsysCgroupPath, "cpu.shares"), []byte(res.CpuShare), 0644)
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
	subsysCgroupPath, err := GetCgroupPat(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	} else {
		// 删除 cgroup 便是删除对应的 cgroupPath 目录
		return os.RemoveAll(subsysCgroupPath)
	}
}

func (s *CpuSubSystem) Apply(cgroupPath string, pid int) error {
	subsysCgroupPath, err := GetCgroupPat(s.Name(), cgroupPath, false)
	if err != nil {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	} else {
		err := os.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644)
		if err != nil {
			return fmt.Errorf("set cgroup proc fail %v", err)
		}
		return nil
	}
}

func (s *CpuSubSystem) Name() string {
	return "cpu"
}
