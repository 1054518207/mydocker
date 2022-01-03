package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"syscall"
)

func RunContainerInitProcess(command string, args []string) error {
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
