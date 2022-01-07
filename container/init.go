package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func runContainerInitProcess(command string, args []string) error {
	/*
		这里的init函数执行是在容器内部的，也就是说，代码执行到这里后，其实容器所在的进程已经创建出来了，我们是本容器执行的第一个进程。
		1.由于现在使用systemd，mount namespace 就变成 shared by default, 所以在挂载 proc文件系统之前需要显式说明让 mount namespace 独立
		2.使用 mount 挂载proc 文件系统，方便我们通过ps等系统命令去查看当前进程资源情况
	*/
	logrus.Infof("command: %s", command)

	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示声明这个新的mount namespace独立
	_ = syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	/*
		MS_PRIVATE：Make this mount private. Mount and unmount events do not propagate into or out of this mount.
					此系统调用使得创建private挂载方式
		MS_REC (since Linux 2.4.11)： Used in conjunction with MS_BIND to create a recursive bind mount, and in
				conjunction with the propagation type flags to recursively change the propagation type of all the
				mounts in a subtree. 此系统调用用于更改当前namespace的进程调用树
	*/

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	/*
	   MS_NOEXEC 在本文件系统中不允许运行其他程序
	   MS_NOSUID 在本系统中运行程序的时候不允许set-user-ID或者set-group-ID
	   MS_NODEV 这个参数是自从Linux 2.4以来所有 mount 的系统都会默认设定的参数
	*/
	err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logrus.Error("syscall mount error")
		return err
	}
	argv := []string{command}
	// 调用这个方法，将用户指定的进程运行起来，把最初的 init 进程给替换掉，当我们进入到容器内部的时候，发现容器内的第一个程序就是我们指定的进程
	if err := syscall.Exec(command, argv, os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
}

func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("Run container get user command error, cmdArray is nil")
	}

	testSetUpMount()
	//setUpMount()

	// LookPath在环境变量中查找可执行二进制文件，如果file中包含一个斜杠，则直接根据绝对路径或者相对本目录的相对路径去查找
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		logrus.Errorf("Exec look path error %v", err)
		return err
	}
	logrus.Infof("Find path %v", path)
	err = syscall.Exec(path, cmdArray[0:], os.Environ())
	if err != nil {
		logrus.Errorf(err.Error())
		return err
	}
	return nil
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

func testSetUpMount() {
	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示声明这个新的mount namespace独立
	_ = syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	/*
		MS_PRIVATE：Make this mount private. Mount and unmount events do not propagate into or out of this mount.
					此系统调用使得创建private挂载方式
		MS_REC (since Linux 2.4.11)： Used in conjunction with MS_BIND to create a recursive bind mount, and in
				conjunction with the propagation type flags to recursively change the propagation type of all the
				mounts in a subtree. 此系统调用用于更改当前namespace的进程调用树
	*/

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	/*
	   MS_NOEXEC 在本文件系统中不允许运行其他程序
	   MS_NOSUID 在本系统中运行程序的时候不允许set-user-ID或者set-group-ID
	   MS_NODEV 这个参数是自从Linux 2.4以来所有 mount 的系统都会默认设定的参数
	*/
	err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logrus.Error("syscall mount error")
	}
}

/**
Init 挂载点
*/
func setUpMount() {
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Get current location error %v", err)
		return
	}
	logrus.Infof("Current location is %s", pwd)
	_ = pivotRoot(pwd)

	//mount proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	_ = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	_ = syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
}

func pivotRoot(root string) error {
	/**
	  为了使当前root的老 root 和新 root 不在同一个文件系统下，我们把root重新mount了一次
	  bind mount是把相同的内容换了一个挂载点的挂载方法
	*/
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("Mount rootfs to itself error: %v", err)
	}
	// 创建 rootfs/.pivot_root 存储 old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}
	// pivot_root 到新的rootfs, 现在老的 old_root 是挂载在rootfs/.pivot_root
	// 挂载点现在依然可以在mount命令中看到
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %v", err)
	}
	// 修改当前的工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %v", err)
	}

	pivotDir = filepath.Join("/", ".pivot_root")
	// umount rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %v", err)
	}
	// 删除临时文件夹
	return os.Remove(pivotDir)
}
