package main

import (
	"fmt"
	"mydocker/container"
	"mydocker/util"
	"os/exec"
	"path"

	"github.com/sirupsen/logrus"
)

func commitContainer(containerId string) {
	imageName := containerId
	mntURL := fmt.Sprintf(container.AUFSRootUrl, containerId)
	mntURL = path.Join(mntURL, container.AUFSMountLayer)
	exist, err := util.FileOrDirExits(mntURL)
	if err != nil {
		logrus.Errorf("mntUrl %s judge error %v", mntURL, err)
		return
	}
	if !exist {
		logrus.Errorf("mnt url %s not found", mntURL)
		return
	}
	imageTar := "/root/" + imageName + ".tar"
	fmt.Printf("%s", imageTar)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		logrus.Errorf("Tar folder %s error %v", mntURL, err)
		return
	}
}
