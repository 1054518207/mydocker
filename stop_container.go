package main

import (
	"encoding/json"
	"fmt"
	"mydocker/cgroups"
	"mydocker/container"
	"mydocker/util"
	"os"
	"path"
	"strconv"

	"github.com/sirupsen/logrus"
)

func StopContainer(containerId string) {
	c, err := container.GetContainerInfoById(containerId)
	if err != nil {
		logrus.Errorf("can not get container info %s error %v", containerId, err)
		return
	}
	if c == nil {
		logrus.Errorf("can not get container info %s , container is null pointer, error %v", containerId, err)
		return
	}
	if c.Status != container.RUNNING {
		logrus.Errorf("can not stop a not running container, container ID = %s error %v", containerId, err)
		return
	}
	pid := c.Pid
	pidInt, err := strconv.Atoi(pid)
	if err != nil{
		logrus.Errorf("can not convert Pid %s into integer error %v", pid, err)
		return
	}
	if err := util.KillProcess(pidInt); err != nil {
		logrus.Errorf("stop a running container %s error %v", containerId, err)
		return
	}
	c.Status = container.STOP
	c.Pid = " "
	contentBytes, err := json.Marshal(c)
	if err != nil{
		logrus.Errorf("Json marshal %s error %v", containerId, err)
		return
	}
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	configFile := path.Join(dirUrl, container.ConfigName)
	// 重新写入配置文件
	if err := os.WriteFile(configFile, contentBytes, 0622); err != nil {
		logrus.Errorf("error write to config file %s error %v", configFile, err)
	}
	// 删除 cgroup 部分，如果restart需要重新写入cgroup
	cgroupManager := cgroups.CgroupManager{Path: fmt.Sprintf(container.CGroup, containerId)}
	cgroupManager.Destroy()
	// 删除 AUFS 挂载
	rootURL := fmt.Sprintf(container.AUFSRootUrl, c.Id)
	mntURL := path.Join(rootURL, container.AUFSMountLayer)
	volume := c.Volume
	// 此处先不考虑 volume 部分，后续等多容器操作时补全volume挂载部分
	container.DeleteAUFSWorkSpace(rootURL, mntURL, volume)
}