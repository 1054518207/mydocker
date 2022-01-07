package subsystems

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"strconv"
)

// MemorySubsystem memory subsystem 的实现
type MemorySubSystem struct {
}

// Set 设置 cgroupPath 对应的 cgroup 内存资源限制
func (s *MemorySubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPat(s.Name(), cgroupPath, true); err == nil {
		if res.MemoryLimit != "" {
			err := os.WriteFile(path.Join(subsysCgroupPath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), 0644)
			if err != nil {
				return fmt.Errorf("set cgroup memory fail %v", err)
			}
		}
		return nil
	} else {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
}

func (s *MemorySubSystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := GetCgroupPat(s.Name(), cgroupPath, false)
	logrus.Infof("Try to remove memory subsystem, remove dir: %v", subsysCgroupPath)
	if err != nil {
		return err
	} else {
		// 删除 cgroup 便是删除对应的 cgroupPath 目录
		return os.RemoveAll(subsysCgroupPath)
	}
}

func (s *MemorySubSystem) Apply(cgroupPath string, pid int) error {
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

func (s *MemorySubSystem) Name() string {
	return "memory"
}
