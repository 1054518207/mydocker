package util

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"syscall"
	"time"
)

func FileOrDirExits(name string) (bool, error) {
	// https://studygolang.com/articles/5435
	// 如果返回的错误为nil,说明文件或文件夹存在
	// 如果返回的错误类型使用os.IsNotExist()判断为true,说明文件或文件夹不存在
	// 如果返回的错误为其它类型,则不确定是否在存在
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) { // 不存在
		return false, nil
	} else {
		return false, err
	}

}

func KillProcess(pid int) error {
	processDir := fmt.Sprintf("/proc/%d", pid)
	if exits, _ := FileOrDirExits(processDir); !exits {
		return nil
	}
	msg := "failed to kill the process %d by sending signal %s"
	err := syscall.Kill(pid, syscall.SIGTERM) // 给进程发送终止信号
	if err != nil {
		logrus.Warnf(msg, pid, "SIGTERM")
		time.Sleep(50 * time.Millisecond)
		// 等待 50 ms，后面直接kill进程
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			return fmt.Errorf(msg, pid, "SIGKILL")
		}
	}
	return nil
}
