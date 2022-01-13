package container

import (
	"github.com/sirupsen/logrus"
	"mydocker/util"
	"os"
	"os/exec"
	"path"
)

const (
	baseLayer    = "busybox"
	baseLayerTar = "busybox.tar"
	writeLayer   = "writeLayer"
)

func NewAUFSWorkSpace(rootURL, mntURL, volume string) {
	CreateReadonlyLayer(rootURL)
	CreateWriteLayer(rootURL)
	CreateMountPoint(rootURL, mntURL)
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(rootURL, mntURL, volumeURLs)
			logrus.Infof("VolumeURLs: %v", volumeURLs)
		} else {
			logrus.Warnf("Volume parameter input is invalid.")
		}
	}
}

// CreateReadonlyLayer 将busybox.tar解压到busybox目录下，作为容器的只读层
func CreateReadonlyLayer(rootURL string) {
	busyBoxURL := path.Join(rootURL, baseLayer)
	busyBoxTarURL := path.Join(rootURL, baseLayerTar)
	exits, err := util.FileOrDirExits(busyBoxURL)
	if err != nil {
		logrus.Warnf("Fail to judge whether dir %s exists. %v", busyBoxURL, err)
	}
	if !exits {
		err := os.MkdirAll(busyBoxURL, os.ModePerm)
		if err != nil {
			logrus.Errorf("Mkdir dir %s error. %v", busyBoxURL, err)
			return
		}
		// CombinedOutput runs the command and returns its combined standard output and standard error 立即运行
		_, err = exec.Command("tar", "-xvf", busyBoxTarURL, "-C", busyBoxURL).CombinedOutput()
		if err != nil {
			logrus.Errorf("untar dir %s error. %v", busyBoxTarURL, err)
			return
		}
	}
}

// CreateWriteLayer 创建一个名称为 writeLayer 的目录作为容器的唯一可写层
func CreateWriteLayer(rootURL string) {
	writeURL := path.Join(rootURL, writeLayer)
	err := os.MkdirAll(writeURL, os.ModePerm)
	if err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", writeURL, err)
		return
	}
}

func CreateMountPoint(rootURL, mntURL string) {
	// 创建 mnt 目录作为挂载点
	err := os.MkdirAll(mntURL, os.ModePerm)
	if err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", mntURL, err)
		return
	}

	// 把 writeLayer 目录和 baseLayer 目录 mount 到mnt目录下
	// dirs 指定的左边起第一个目录是 read-write 权限，后续目录都是 read-only 权限
	// 由于 aufs 是虚拟文件系统，挂载点设置为 none
	// https://www.cnblogs.com/sparkdev/p/11237347.html
	dirs := "dirs=" + path.Join(rootURL, writeLayer) + ":" + path.Join(rootURL, baseLayer)
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mount aufs error, mount command: mount -t aufs -o %v none %v. err: %v", dirs, mntURL, err)
		return
	}
}

func DeleteAUFSWorkSpace(rootURL, mntURL string, volume string) {
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			DeleteMountPointWithVolume(rootURL, mntURL, volumeURLs)
		} else {
			DeleteMountPoint(mntURL)
		}
	} else {
		DeleteMountPoint(mntURL)
	}
	DeleteWriteLayer(rootURL)
}

func DeleteMountPoint(mntURL string) {
	cmd := exec.Command("umount", mntURL)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Unmount dir %s error %v", mntURL, err)
		return
	}
	if err := os.RemoveAll(mntURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", mntURL, err)
		return
	}
}

func DeleteWriteLayer(rootURL string) {
	writeURL := path.Join(rootURL, writeLayer)
	if err := os.RemoveAll(writeURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", writeURL, err)
		return
	}
}
