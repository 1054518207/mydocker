package main

import (
	"fmt"
	"io"
	"mydocker/container"
	"os"
	"path"

	"github.com/sirupsen/logrus"
)

func LogContainer(containerId string){
	// 找到对应的目录位置
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	logFileLocation := path.Join(dirUrl, container.LogFileName)
	// 打开并读取日志文件
	file, err := os.Open(logFileLocation)
	if err != nil{
		logrus.Errorf("Log file open %s error %v", logFileLocation, err)
		return
	}
	defer file.Close()
	contentBytes, err := io.ReadAll(file)
	if err != nil{
		logrus.Errorf("Log file read %s error %v", logFileLocation, err)
		return
	}
	// 使用 fmt.Fprint 函数将读出的内容输出到标准输出上，此方式可以不更改原先设置的输出流
	fmt.Fprint(os.Stdout, string(contentBytes))
}
