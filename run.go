package main

import (
	"fmt"
	"mydocker/cgroups"
	"mydocker/cgroups/subsystems"
	"mydocker/container"
	"mydocker/util"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig, volume, containerName string, envSlice []string) {

	// generate 10 bits random container ID
	containerId := util.RandStringBytes(10)

	parent, writePipe := container.NewParentProcess(tty, volume, containerId, envSlice)
	if parent == nil {
		logrus.Errorf("new parent process error")
		return
	}
	
	if err := parent.Start(); err != nil {
		logrus.Error(err)
		return
	}

	// record container info
	_, err := container.RecordContainerInfo(parent.Process.Pid, comArray, containerName, containerId, volume)
	if err != nil{
		logrus.Errorf("Record container info error %v", err)
		return
	}

	// use mydocker-cgroup as cgroup name
	cgroupManager := cgroups.CgroupManager{Path: fmt.Sprintf(container.CGroup, containerId)}
	// 设置资源限制
	_ = cgroupManager.Set(res)
	// 将容器进程加入到各个subsystem挂载对应的cgroup中
	_ = cgroupManager.Apply(parent.Process.Pid)
	// 对容器设置完限制后，初始化容器
	sendInitCommand(comArray, writePipe)

	if tty {
		_ = parent.Wait()
		// 删除 AUFS 挂载
		rootURL := fmt.Sprintf(container.AUFSRootUrl, containerId)
		mntURL := path.Join(rootURL, container.AUFSMountLayer)
		container.DeleteAUFSWorkSpace(rootURL, mntURL, volume)
		container.DeleteContainerInfo(containerId)
		cgroupManager.Destroy()
	}

}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	logrus.Infof("command all is %s", command)
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close()
}
