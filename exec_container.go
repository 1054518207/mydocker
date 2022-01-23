package main

import (
	"encoding/json"
	"fmt"
	"mydocker/container"
	"mydocker/util"
	_ "mydocker/nsenter"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

// 此部分常量要与 nsenter.go 里面的设置相同
const ENV_EXEC_PID = "container_pid"
const ENV_EXEC_CMD = "container_cmd"

func ExecContainerCommand(containerId string, cmdArr []string) error{
	pid, err := getContainerPidById(containerId)
	if err != nil{
		logrus.Errorf("can not get pid from containerId = %s error %v", containerId, err)
		return err
	}
	// 拼接exec命令
	command := strings.Join(cmdArr, " ")
	logrus.Infof("Get container Pid = %s", pid)
	logrus.Infof("Get container command = %s", command)

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, command)

	// 获取容器内环境变量，并赋值到exec命令中
	containerEnvs := container.GetEnvByPid(pid)
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err := cmd.Run(); err != nil {
		logrus.Errorf("Exec containerId = %s error %v", containerId, err)
		return err
	}

	return nil
}

func getContainerPidById(containerId string) (string, error) {
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	configFile := path.Join(dirUrl, container.ConfigName)
	exist, err := util.FileOrDirExits(configFile)
	if err != nil{
		logrus.Errorf("can not find config file %s error %v", configFile, err)
		return "", err
	}
	if !exist {
		logrus.Errorf("no config file in %s", configFile)
		return "", fmt.Errorf("no config file in %s", configFile)
	}
	contentBytes, err := os.ReadFile(configFile)
	if err != nil{
		logrus.Errorf("read config file %s error %v", configFile, err)
		return "", err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		logrus.Errorf("parse config file %s error %v", configFile, err)
		return "", err
	}
	return containerInfo.Pid, nil
}