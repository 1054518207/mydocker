package main

import (
	"github.com/sirupsen/logrus"
	"mydocker/cgroups"
	"mydocker/cgroups/subsystems"
	"mydocker/container"
	"os"
	"strings"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig) {
	parent, writePipe := container.NewParentProcess(tty)
	if parent == nil {
		logrus.Errorf("new parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		logrus.Error(err)
		return
	}
	// use mydocker-cgroup as cgroup name
	cgroupManager := cgroups.CgroupManager{Path: "mydocker-cgroup"}
	defer func(cgroupManager *cgroups.CgroupManager) {
		// destroy cgroup after exit container
		err := cgroupManager.Destroy()
		if err != nil {
			logrus.Error(err)
		}
	}(&cgroupManager)
	// 设置资源限制
	_ = cgroupManager.Set(res)
	// 将容器进程加入到各个subsystem挂载对应的cgroup中
	_ = cgroupManager.Apply(parent.Process.Pid)
	// 对容器设置完限制后，初始化容器
	sendInitCommand(comArray, writePipe)
	_ = parent.Wait()
	// 删除 AUFS 挂载
	defer func() {
		mntURL := "/root/mnt"
		rootURL := "/root"
		container.DeleteAUFSWorkSpace(rootURL, mntURL)
	}()
}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	logrus.Infof("command all is %s", command)
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close()
}
