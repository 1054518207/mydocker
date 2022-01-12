package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"syscall"
)

func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("run container get user command error, cmdArray is nil")
	}

	_ = setupMount()

	// LookPath在环境变量中查找可执行二进制文件，如果file中包含一个斜杠，则直接根据绝对路径或者相对本目录的相对路径去查找
	pth, err := exec.LookPath(cmdArray[0])
	if err != nil {
		logrus.Errorf("Exec look path error %v", err)
		return err
	}
	logrus.Infof("Find path %v", pth)
	// 调用这个方法，将用户指定的进程运行起来，把最初的 init 进程给替换掉，当我们进入到容器内部的时候，发现容器内的第一个程序就是我们指定的进程
	err = syscall.Exec(pth, cmdArray[0:], os.Environ())
	if err != nil {
		logrus.Errorf(err.Error())
		return err
	}
	return nil
}

func setupMount() error {
	pwd, _ := os.Getwd()
	logrus.Infof("Current location is '%s', this path will be rootfs.", pwd)
	_ = pivotRoot(pwd)

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	/*
	   MS_NOEXEC 在本文件系统中不允许运行其他程序
	   MS_NOSUID 在本系统中运行程序的时候不允许set-user-ID或者set-group-ID
	   MS_NODEV 这个参数是自从Linux 2.4以来所有 mount 的系统都会默认设定的参数
	*/
	// 挂载 proc，使得容器进程的proc只显示当前进程的信息，否则会显示父进程proc信息
	err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		return fmt.Errorf("syscall mount proc error, err: %v", err)
	}
	// 单独配置设备挂载，隔离父进程设备
	err = syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	if err != nil {
		return fmt.Errorf("syscall mount tmpfs error, err: %v", err)
	}

	return nil
}

func pivotRoot(rootfs string) error {

	// Make oldroot rprivate to make sure our unmounts don't propogate to the host (and thus bork the machine).
	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示声明这个新的mount namespace独立
	// 使后续挂载操作在容器进程退出后不影响原主进程
	// note: runc use the flags MS_SLAVE and MS_REC.
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to enable the mount namespace work properly: %v", err)
	}
	/*
		MS_PRIVATE：Make this mount private. Mount and unmount events do not propagate into or out of this mount.
					此系统调用使得创建private挂载方式
		MS_REC (since Linux 2.4.11)： Used in conjunction with MS_BIND to create a recursive bind mount, and in
				conjunction with the propagation type flags to recursively change the propagation type of all the
				mounts in a subtree. 此系统调用用于更改当前namespace的进程调用树
	*/

	// 使当前rootfs的老root和新root不在同一个文件夹下，需要把rootfs重新mount一次，bind mount 是把相同的内容换一个挂载点的挂载方法
	if err := syscall.Mount(rootfs, rootfs, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to mount rootfs to itself: %v", err)
	}

	// 创建 rootfs/.pivot_root存储 old_root
	pivotDir := path.Join(rootfs, ".oldroot")
	if err := os.Mkdir(pivotDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to mkdir old_root %s: %v", pivotDir, err)
	}

	// pivot_root 到新的rootfs，老的old_root现在挂载在 rootfs/.pivot_root上，挂载点目前依然可以在 mount 命令中看到
	// https://man7.org/linux/man-pages/man2/pivot_root.2.html
	// PivotRoot的另外一种调用方式：https://github.com/opencontainers/runc/commit/f8e6b5af5e120ab7599885bd13a932d970ccc748
	if err := syscall.PivotRoot(rootfs, pivotDir); err != nil {
		return fmt.Errorf("failed to syscall pivot_root: %v", err)
	}

	// 搭配 PivotRoot 使用，修改当前的工作目录到根目录 pivot_root() does not change the
	// caller's current working directory (unless it is on the old root
	// directory), and thus it should be followed by a chdir("/") call.
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("failed to syscall chdir /: %v", err)
	}

	pivotDir = path.Join("/", ".oldroot")
	// unmount rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("failed to unmount old root dir: %v", err)
	}

	// note: need to delete the origin directory /dev
	for _, dir := range []string{pivotDir, "/dev"} {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}

	return nil
}
