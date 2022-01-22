package container

import (
	"encoding/json"
	"fmt"
	"io"
	"mydocker/util"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

func newPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}

func readUserCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := io.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

func GetContainerInfoById (containerId string) (*ContainerInfo, error) {
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerId)
	configFile := path.Join(dirUrl, ConfigName)
	exists, err := util.FileOrDirExits(configFile)
	if err != nil {
		logrus.Errorf("error judge file or dir %s exits error %v", dirUrl, err)
		return nil, err
	}
	if !exists {
		logrus.Errorf("no config file %s error", configFile)
		return nil, fmt.Errorf("no config file %s found", configFile)
	}
	contentBytes, err := os.ReadFile(configFile)
	if err != nil {
		logrus.Errorf("error read config file %s exits error %v", configFile, err)
		return nil, err
	}
	var containerInfo ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo) ; err != nil {
		logrus.Errorf("error get unmarshal json config file %s error %v", configFile, err)
		return nil, err
	}
	return &containerInfo, nil
}