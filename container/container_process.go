package container

import (
	"fmt"
	"mydocker/util"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/sirupsen/logrus"
)

func NewParentProcess(tty bool, volume, containerId string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := newPipe()
	if err != nil {
		logrus.Errorf("New pipe error %v", err)
		return nil, nil
	}
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else { // detach mode
		logDirUrl := fmt.Sprintf(DefaultInfoLocation, containerId)
		if err := os.MkdirAll(logDirUrl, 0644); err != nil{
			logrus.Errorf("New Parent Process mkdir %s error %v", logDirUrl, err)
			return nil, nil
		}
		containerLogFile := path.Join(logDirUrl, LogFileName)
		var logFile *os.File
		exist, err := util.FileOrDirExits(containerLogFile)
		if err != nil {
			logrus.Error("Error when find tmpLogFile")
			return nil, nil
		}
		if exist {
			logFile, err = os.OpenFile(containerLogFile, syscall.O_WRONLY|syscall.O_APPEND, 0644)
			if err != nil {
				logrus.Error("Error when open tmpLogFile")
				return nil, nil
			}
		} else {
			logFile, err = os.Create(containerLogFile)
			if err != nil {
				logrus.Error("Error when create tmpLogFile")
				return nil, nil
			}
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	cmd.ExtraFiles = []*os.File{readPipe}

	mntURL := "/root/mnt"
	rootURL := "/root"
	NewAUFSWorkSpace(rootURL, mntURL, volume)
	// 配置 rootfs
	cmd.Dir = mntURL

	// cmd.Dir = "/root/busybox"
	return cmd, writePipe
}
