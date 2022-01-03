package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func NewParentProcess(tty bool, command string) *exec.Cmd {
	/*
		这里是父进程也就是我们当前进程执行的内容，根据我们上一章介绍的内容，应该比较容易明白
		1.这里的/proc/self/exe 调用，其中/proc/self指的是当前运行进程自己的环境，exe其实就是自己调用了自己，我们使用这种方式实现对创建出来的进程进行初始化
		2.后面args是参数，其中 init 是传递给本进程的第一个参数，这在本例子中，其实就是会去调用我们的initCommand去初始化进程的一些环境和资源
		3.下面的clone 参数就是去 fork 出来的一个新进程，并且使用了namespace隔离新创建的进程和外部的环境。
		4.如果用户指定了-ti 参数，我们就需要把当前进程的输入输出导入到标准输入输出上
	*/
	args := []string{"init", command}
	fmt.Println(args)
	cmd := exec.Command("/proc/self/exe", args...)
	// ProcAttr holds the attributes that will be applied to a new process
	cmd.SysProcAttr = &syscall.SysProcAttr{Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID |
		syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd
}
