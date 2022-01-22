package container

import (
	"encoding/json"
	"fmt"
	"mydocker/util"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type ContainerInfo struct {
	Pid        string `json:"pid"`
	Id         string `json:"Id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	CreateTime string `json:"createTime"`
	Status     string `json:"status"`
	Volume     string `json:"volume"`
}

var (
	RUNNING             = "running"
	STOP                = "stop"
	Exit                = "exit"
	DefaultInfoLocation = "/var/run/mydocker/%s/"
	ConfigName          = "config.json"
	LogFileName         = "container.log"

	// AUFS 配置
	AUFSRootUrl         = "/var/run/mydocker/%s/"
	AUFSWriteLayer      = "writerlayer"
	AUFSMountLayer      = "mnt"
	BusyBoxUrl			= "/root/busybox"
	BusyBoxTarUrl       = "/root/busybox.tar"

	// cgroup配置
	CGroup = "mydocker-cgroup/%s"
)

func RecordContainerInfo(containerPID int, cmdArr []string, containerName, id, volume string) (string, error) {

	// current time is container create time
	createTime := time.Now().Format(util.TIMESTAP)
	command := strings.Join(cmdArr, " ")
	// default name is id
	if containerName == "" {
		containerName = id
	}
	// generate struct instance
	containerInfo := &ContainerInfo{
		Id:         id,
		Pid:        strconv.Itoa(containerPID),
		Name:       containerName,
		Command:    command,
		CreateTime: createTime,
		Status:     RUNNING,
		Volume:     volume,
	}
	// 将容器信息对象json序列化为字符串
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		logrus.Errorf("Record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)

	// get container info save dir 使用容器id作为路径信息
	dirUrl := fmt.Sprintf(DefaultInfoLocation, id)
	// if dir not exist, we will create automatically
	if err := os.MkdirAll(dirUrl, 0644); err != nil {
		logrus.Errorf("Record container info error %v", err)
		return "", err
	}
	configFile := path.Join(dirUrl, ConfigName)
	file, err := os.Create(configFile)

	defer func() {
		err := file.Close()
		if err != nil {
			logrus.Errorf("File close error %v", err)
		}
	}()

	if err != nil {
		logrus.Errorf("File create error %v", err)
		return "", err
	}

	if _, err := file.WriteString(jsonStr); err != nil {
		logrus.Errorf("File write json string error %v", err)
		return "", err
	}
	return id, nil
}

func DeleteContainerInfo(containerId string) {
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerId)
	if err := os.RemoveAll(dirUrl); err != nil {
		logrus.Errorf("Remove dir %s error %v", dirUrl, err)
	}
}
