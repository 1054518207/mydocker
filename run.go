package main

import (
	"awesomeProject/container"
	"github.com/sirupsen/logrus"
	"os"
)

func Run(tty bool, command string) {
	/*
		这里的Start方法是真正开始前面创建好的command的调用，他会首先 clone
		出来一个 namespace 隔离的进程，然后在子进程中，调用/proc/self/exe 也就是自己，发送 init 参数，调用我们写的init方法，去初始化容器的一些资源
	*/
	parent := container.NewParentProcess(tty, command)
	if err := parent.Start(); err != nil {
		logrus.Errorf(err.Error())
	}
	err := parent.Wait()
	if err != nil {
		logrus.Errorf(err.Error())
		return
	}
	os.Exit(-1)
}
