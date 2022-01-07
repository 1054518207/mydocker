package subsystems

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
)

// FindCgroupMountpoint 通过 /proc/self/mountinfo 找出挂载了某个 subsystem 的 hierarchy cgroup 根节点所在的目录
// eg: 38 33 0:33 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime shared:15 - cgroup cgroup rw,memory
// 返回 /sys/fs/cgroup/memory
func FindCgroupMountpoint(subsystem string) string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Println("error when clos f:", err)
		}
	}(f)

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			if opt == subsystem {
				logrus.Infof("Find subsystem: %v, fields: %v", subsystem, fields[len(fields)-1])
				return fields[4]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}

// GetCgroupPat 得到 cgroup 在文件系统中的绝对路径，即获取当前subsystem在虚拟文件系统中的路径
func GetCgroupPat(subsystem string, cgroupPath string, autoCreate bool) (string, error) {

	cgroupRoot := FindCgroupMountpoint(subsystem)

	// https://studygolang.com/articles/5435
	// 如果返回的错误为nil,说明文件或文件夹存在
	// 如果返回的错误类型使用os.IsNotExist()判断为true,说明文件或文件夹不存在
	// 如果返回的错误为其它类型,则不确定是否在存在
	if _, err := os.Stat(path.Join(cgroupRoot, cgroupPath)); err != nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path.Join(cgroupRoot, cgroupPath), 0755); err == nil {
				logrus.Infof("Mkdir successfully, new dir: %v", path.Join(cgroupRoot, cgroupPath))
			} else {
				return "", fmt.Errorf("error create cgroup %v", err)
			}
		}
		logrus.Infof("Current subsystem: %v, work in dir: %v", subsystem, path.Join(cgroupRoot, cgroupPath))
		return path.Join(cgroupRoot, cgroupPath), nil
	} else {
		return "", fmt.Errorf("cgroup path error %v", err)
	}

}
