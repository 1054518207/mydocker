package subsystems

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"mydocker/util"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	cgroup      = "cgroup"
	cgroupProcs = "cgroup.procs"
	mountInfo   = "/proc/self/mountinfo"
)

// FindCgroupMountpoint 通过 /proc/self/mountinfo 找出挂载了某个 subsystem 的 hierarchy cgroup 根节点所在的目录
// 42 33 0:37 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime shared:19 - cgroup cgroup rw,memory
// 返回 /sys/fs/cgroup/memory
func FindCgroupMountpoint(subsystemName string) string {
	f, err := os.Open(mountInfo)
	if err != nil {
		return ""
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Println("error when close f:", err)
		}
	}(f)

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		// 40 33 0:35 / /sys/fs/cgroup/cpu,cpuacct rw,nosuid,nodev,noexec,relatime shared:17 - cgroup cgroup rw,cpu,cpuacct
		// 42 33 0:37 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime shared:19 - cgroup cgroup rw,memory
		txt := scanner.Text() // a line
		fields := strings.Split(txt, " ")
		if fields[8] == cgroup && fields[9] == cgroup && strings.HasSuffix(fields[4], subsystemName) {
			return fields[4]
		}
	}

	if err := scanner.Err(); err != nil {
		return ""
	}

	return ""
}

// GetCgroupPath GetCgroupPat 得到 cgroup 在文件系统中的绝对路径，即获取当前subsystem在虚拟文件系统中的路径
// cgroupPath: /sys/fs/cgroup/memory/{cgroupPath}
func GetCgroupPath(subsystem string, cgroupPath string) (string, error) {
	cgroupRoot := FindCgroupMountpoint(subsystem)

	// 保证dir存在,MkdirAll会创建一个名为path的目录以及任何必要的父项，并返回nil，否则返回错误。
	// 许可位perm用于MkdirAll创建的所有目录。如果path已经是一个目录，MkdirAll什么也不做，并返回nil。
	if err := os.MkdirAll(path.Join(cgroupRoot, cgroupPath), 0755); err != nil {
		return "", fmt.Errorf("failed to mkdir %s: %v", path.Join(cgroupRoot, cgroupPath), err)
	}
	return path.Join(cgroupRoot, cgroupPath), nil

}

func apply(subsystemName string, cgroupPath string, pid int) error {
	subsystemPath, err := GetCgroupPath(subsystemName, cgroupPath)
	if err != nil {
		return err
	}
	targetFile := path.Join(subsystemPath, cgroupProcs)
	targetVal := []byte(strconv.Itoa(pid))
	err = os.WriteFile(targetFile, targetVal, 0644)
	if err != nil {
		return fmt.Errorf("fail to add process %d into subsystem %s, err info: %v", pid, subsystemName, err)
	}
	return nil
}

func remove(subsystemName, cgroupPath string) error {
	subsystemPath, err := GetCgroupPath(subsystemName, cgroupPath)
	if err != nil {
		return err
	}
	cgroupProcsFile := path.Join(subsystemPath, cgroupProcs)
	procsBytes, err := os.ReadFile(cgroupProcsFile)
	if err != nil {
		return err
	}

	if len(procsBytes) != 0 {
		// notes: if the cgroupProcs file of container's cgroup is still NOT empty,
		// which maybe contain zombie processes, we must move these processes to the
		// container's parent cgroup of current subsystem before rmdir it.
		logrus.Debugf("the contents of %s is still NOT empty", cgroupProcsFile)
		procsString := string(procsBytes[:len(procsBytes)-1]) // 测试所得
		parentCgroupProcs := path.Join(path.Dir(subsystemPath), cgroupProcs)
		// NOTE: can only add ONE process to the file cgroup.procs once.
		processArr := strings.Split(procsString, "\n")
		for i := len(processArr) - 1; i >= 0; i-- {
			processStr := processArr[i]
			processId, _ := strconv.Atoi(processStr)
			// 在主程序退出前需要 kill 主程序的子进程
			_ = util.KillProcess(processId)
			time.Sleep(100 * time.Millisecond)

			// 检查是否目标进程已被kill，如果被kill则继续，否则当作可以restart的进程
			// 对于可以restart的进程，考虑将其写入父 cgroup
			processDir := fmt.Sprintf("/proc/%d", processId)
			if exits, _ := util.FileOrDirExits(processDir); !exits {
				continue
			}

			logrus.Debugf("moving the orphan process %s to its parent cgroup %s", processStr, parentCgroupProcs)
			err := os.WriteFile(parentCgroupProcs, []byte(processStr), 0644)
			if err != nil {
				return err
			}
		}
	}
	// 注意，无法将残留cgroup目录中的文件删除，只能将容器名称的目录直接删除
	logrus.Infof("removing %s", subsystemPath)
	// `os.RemoveAll(subsystemPath)` doesn't work!
	// `exec.Command("rmdir", subsystemPath).Run()` is also ok.
	return os.Remove(subsystemPath)
}
