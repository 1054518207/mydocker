package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"strings"
)

// 解析 volume 字符串
func volumeUrlExtract(volume string) []string {
	var volumeURLs []string
	volumeURLs = strings.Split(volume, ":")
	return volumeURLs
}

// MountVolume 挂载数据卷
// 1. 读取宿主机文件目录URL，创建宿主机文件目录（/root/{parentUrl}）
// 2. 读取容器挂载点URL，在容器文件系统里创建挂载点(/root/mnt/${containerUrl})
// 3. 把宿主机文件目录挂载到容器挂载点，启动容器过程，同时对数据卷的处理也随之运行
func MountVolume(rootURL, mntURL string, volumeURLs []string) {
	// create host dir
	parentUrl := volumeURLs[0]
	if err := os.MkdirAll(parentUrl, os.ModePerm); err != nil {
		logrus.Errorf("Mkdir parent dir %s err. %v", parentUrl, err)
		return
	}
	// 在容器文件系统中创建挂载点
	containerUrl := volumeURLs[1]
	containerVolumeURL := path.Join(mntURL, containerUrl)
	if err := os.MkdirAll(containerVolumeURL, os.ModePerm); err != nil {
		logrus.Errorf("Mkdir container dir %s err. %v", containerVolumeURL, err)
		return
	}
	// 把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Mount volume in %s failed. %v", parentUrl, err)
		return
	}
}

func DeleteMountPointWithVolume(rootURL, mntURL string, volumeURLs []string) {
	// 卸载容器文件系统挂载点
	containerUrl := volumeURLs[1]
	containerVolumeURL := path.Join(mntURL, containerUrl)
	cmd := exec.Command("umount", containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Umount volume %s failed. %v", containerVolumeURL, err)
		return
	}

	// 卸载整个容器文件系统的挂载点
	cmd = exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Umount mountpoint %s failed. %v", containerVolumeURL, err)
		return
	}

	// 删除容器文件系统挂载点
	if err := os.RemoveAll(mntURL); err != nil {
		logrus.Warnf("Remove mountpoint dir %s error %v", mntURL, err)
		return
	}
}
