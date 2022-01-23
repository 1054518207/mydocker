package container

import (
	"fmt"
	"mydocker/images"
	"mydocker/util"
	"os"
	"os/exec"
	"path"

	"github.com/sirupsen/logrus"
)

func NewAUFSWorkSpace(rootURL, mntURL, volume, imageName string) error{
	if err := CreateReadonlyLayer(imageName); err != nil{
		return err
	}
	CreateWriteLayer(rootURL)
	readonlyLayer := path.Join(images.ImagesStoreDir, imageName)
	CreateMountPoint(rootURL, mntURL, readonlyLayer )
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			volumeDir := volumeURLs[0]
			containerDir :=  volumeURLs[1]
			writeLayer := path.Join(rootURL, AUFSMountLayer)
			containerDir = path.Join(writeLayer, containerDir)
			CreateVolumeMount(volumeDir, containerDir)
			logrus.Infof("VolumeURLs: %v", volumeURLs)
		} else {
			logrus.Warnf("Volume parameter input is invalid.")
		}
	}
	return nil
}

func CreateVolumeMount(source, target string) error {
	if err := os.MkdirAll(source, 0755); err != nil {
		return fmt.Errorf("failed to mkdir %s: %v", source, err)
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("failed to mkdir container volume dir %s: %v", target, err)
	}
	options := fmt.Sprintf("dirs=%s", source)
	cmd := exec.Command("mount", "-t", "aufs", "-o", options, "none", target)
	logrus.Info("mount volume command: mount", " -t ", " aufs ", " -o ", options, " none ", target)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount local volume: %v", err)
	}
	return nil
}

// CreateReadonlyLayer 将busybox.tar解压到busybox目录下，作为容器的只读层
func CreateReadonlyLayer(imageName string) error{
	// busyBoxURL := BusyBoxUrl
	// busyBoxTarURL := BusyBoxTarUrl
	imageUrl := path.Join(images.ImagesStoreDir, imageName)
	exits, err := util.FileOrDirExits(imageUrl)
	if err != nil {
		return fmt.Errorf("fail to judge whether dir %s exists. %v", imageUrl, err)
	}
	if !exits {
		
		return fmt.Errorf("image not in %s , please create target image URL %s with readonly leayer manually", images.ImagesStoreDir, imageName)

		// err := os.MkdirAll(busyBoxURL, os.ModePerm)
		// if err != nil {
		// 	logrus.Errorf("Mkdir dir %s error. %v", busyBoxURL, err)
		// 	return
		// }

		// CombinedOutput runs the command and returns its combined standard output and standard error 立即运行
		// _, err = exec.Command("tar", "-xvf", busyBoxTarURL, "-C", busyBoxURL).CombinedOutput()
		// if err != nil {
		// 	logrus.Errorf("untar dir %s error. %v", busyBoxTarURL, err)
		// 	return
		// }
	}
	return nil
}

// CreateWriteLayer 创建一个名称为 writeLayer 的目录作为容器的唯一可写层
func CreateWriteLayer(rootURL string) {
	writeURL := path.Join(rootURL, AUFSWriteLayer)
	err := os.MkdirAll(writeURL, os.ModePerm)
	if err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", writeURL, err)
		return
	}
}

func CreateMountPoint(rootURL, mntURL, readonlyURL string){
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
	dirs := "dirs=" + path.Join(rootURL, AUFSWriteLayer) + ":" + readonlyURL
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)
	logrus.Info("mount command : mount", " -t", " aufs", " -o ", dirs, " none ", mntURL)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		logrus.Errorf("mount aufs error, mount command: mount -t aufs -o %v none %v. err: %v", dirs, mntURL, err)
		return
	}
}

func DeleteAUFSWorkSpace(rootURL, mntURL, volume string) {
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			logrus.Infof("volume info: %s", volume)
			containerDir :=  volumeURLs[1]
			writeLayer := path.Join(rootURL, AUFSMountLayer)
			containerDir = path.Join(writeLayer, containerDir)
			UmountVolume(containerDir)
		}
	} 
	DeleteMountPoint(mntURL)
	DeleteWriteLayer(rootURL)
}

func DeleteMountPoint(mntURL string) {
	cmd := exec.Command("umount", mntURL)
	logrus.Info("Umount mount point command: umount ", mntURL)
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
	writeURL := path.Join(rootURL, AUFSWriteLayer)
	if err := os.RemoveAll(writeURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", writeURL, err)
		return
	}
}

func UmountVolume(target string) error {
	cmd := exec.Command("umount", target)
	logrus.Info("Umount volume command: umount ", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Umount volume %s failed. %v", target, err)
		return err
	}
	return nil
}