package cgroups

import (
	"github.com/sirupsen/logrus"
	"mydocker/cgroups/subsystems"
)

type CgroupManager struct {
	// cgroup在hierarchy中的路径 相当于创建的cgroup目录相对于root cgroup目录的路径
	Path string
	// 资源配置
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{Path: path}
}

// Apply 将进程PID加入到每个cgroup中
func (c *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		_ = subSysIns.Apply(c.Path, pid)
	}
	return nil
}

// Set 设置各个subsystem挂载中的cgroup资源限制
func (c *CgroupManager) Set(res *subsystems.ResourceConfig) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		_ = subSysIns.Set(c.Path, res)
	}
	return nil
}

// Destroy 释放各个subsystem挂载中的group
func (c *CgroupManager) Destroy() error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		err := subSysIns.Remove(c.Path)
		if err != nil {
			logrus.Warnf("remove cgroup fail %v, error path: %v", err, c.Path)
		}
	}
	return nil
}
