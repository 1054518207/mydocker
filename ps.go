package main

import (
	"encoding/json"
	"fmt"
	"mydocker/container"
	"os"
	"path"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
)

func ListContainers(){
	// 找到存储容器信息的路径 /var/run/mydocker/%s/
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, "")
	dirUrl = dirUrl[:len(dirUrl)-1]
	// 读取该目录下的所有文件
	dirs, err := os.ReadDir(dirUrl)
	if err != nil{
		logrus.Errorf("Read dir %s error %v", dirUrl, err)
		return
	}

	var containers []*container.ContainerInfo
	for _, container := range dirs{
		containerInfo, err := GetContainerInfo(container)
		if err != nil{
			logrus.Errorf("Get container info error %v", err)
			return
		}
		containers = append(containers, containerInfo)
	}
	// 使用tabwriter.NewWriter在控制台打印信息
	w := tabwriter.NewWriter(os.Stdout,12, 1,3, ' ', 0)
	fmt.Fprint(w, "ID\tName\tPid\tStatus\tCommand\tCreate\n")
	for _, v := range containers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", v.Id, v.Name, v.Pid, v.Status, v.Command, v.CreateTime)
	}
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error ", err)
			return
	}
}

func GetContainerInfo(file os.DirEntry) (*container.ContainerInfo, error) {
	// 获取文件名称
	containerId := file.Name()
	// 根据文件名称生成文件绝对路径
	configFileDir := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	configFileDir = path.Join(configFileDir, container.ConfigName)
	// 读取配置文件信息
	content, err := os.ReadFile(configFileDir)
	if err != nil{
		logrus.Errorf("Read file %s error %v", configFileDir, err)
		return nil, err
	}
	var containerInfo container.ContainerInfo
	// json 信息反序列化为容器信息对象
	err = json.Unmarshal(content, &containerInfo)
	if err != nil{
		logrus.Errorf("json unmarshal error %v", err)
		return nil, err
	}
	return &containerInfo, nil
}