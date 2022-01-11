package subsystems

import (
	"fmt"
	"os"
	"path"
)

const (
	memoryLimit = "memory.limit_in_bytes"
)

// MemorySubsystem memory subsystem 的实现
type MemorySubSystem struct {
}

// Set 设置 cgroupPath 对应的 cgroup 内存资源限制
func (s *MemorySubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	memoryCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath)
	if err == nil {
		if res.MemoryLimit != "" {
			err := os.WriteFile(path.Join(memoryCgroupPath, memoryLimit), []byte(res.MemoryLimit), 0644)
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
	return remove(s.Name(), cgroupPath)
}

func (s *MemorySubSystem) Apply(cgroupPath string, pid int) error {
	return apply(s.Name(), cgroupPath, pid)
}

func (s *MemorySubSystem) Name() string {
	return "memory"
}
