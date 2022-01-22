package main

import (
	"fmt"
	"mydocker/container"
	"os"

	"github.com/sirupsen/logrus"
)

func RemoveContainer(containerId string) {
	c, err := container.GetContainerInfoById(containerId)
	if err != nil {
		logrus.Errorf("can not get container info %s error %v", containerId, err)
		return
	}
	if c == nil {
		logrus.Errorf("can not get container info %s , container is null pointer, error %v", containerId, err)
		return
	}
	if c.Status != container.STOP {
		logrus.Errorf("can not remove a not stopped container, container ID = %s error %v", containerId, err)
		return
	}
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	if err := os.RemoveAll(dirUrl); err != nil {
		logrus.Errorf("error remove %s error %v", dirUrl, err)
		return
	}
	logrus.Infof("remove %s successfully", containerId)
}