package main

import (
	"fmt"
	"mydocker/cgroups/subsystems"
	"mydocker/container"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups limit
			mydocker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "ti", Usage: "enable tty"},
		cli.StringFlag{Name: "m", Usage: "memory limit"},
		cli.StringFlag{Name: "cpushare", Usage: "cpushare limit"},
		cli.StringFlag{Name: "cpuset", Usage: "cpuset limit"},
		cli.StringFlag{Name: "v", Usage: "volume"},
		cli.BoolFlag{Name: "d", Usage: "detach container, run as a daemon"},
		cli.StringFlag{Name: "name", Usage: "Container name"},
	},
	/*
		run命令执行的真正函数
		1. 判断参数是否包含command
		2. 获取用户指定的command
		3. 调用 Run function准备启动容器
	*/
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}
		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}
		tty := context.Bool("ti")
		detach := context.Bool("d")
		volume := context.String("v")
		if tty && detach {
			return fmt.Errorf("ti and d parameter can not be both provided")
		}
		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
			CpuShare:    context.String("cpushare"),
			CpuSet:      context.String("cpuset"),
		}
		logrus.Infof("create tty %v", tty)
		containerName := context.String("name")
		Run(tty, cmdArray, resConf, volume, containerName)
		return nil
	},
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	/*
		1. 获取容器传递过来的 command 参数
		2. 执行容器初始化操作
	*/
	Action: func(ctx *cli.Context) error {
		logrus.Infof("Init come on")
		cmd := ctx.Args().Get(0)
		logrus.Infof("command: %s", cmd)
		err := container.RunContainerInitProcess()
		return err
	},
}

var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		imageName := ctx.Args().Get(0)
		commitContainer(imageName)
		return nil
	},
}

var listCommand = cli.Command{
	Name: "ps",
	Usage: "list all registering containers",
	Action: func (ctx *cli.Context) error {
		ListContainers()
		return nil
	},
}

var logCommand = cli.Command{
	Name: "log",
	Usage: "print log info",
	Action: func (ctx *cli.Context) error  {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container id")
		}
		containerId := ctx.Args().Get(0)
		LogContainer(containerId)
		return nil
	},
}

var execCommand = cli.Command{
	Name: "exec",
	Usage: "exec a command into container",
	Action: func (ctx *cli.Context) error {
		// callback
		if os.Getenv(ENV_EXEC_PID) != "" {
			logrus.Infof("pid callback pid = %d", os.Getgid())
			return nil
		}
		// command format: mydocker exec 容器Id 命令
		if len(ctx.Args()) < 2 {
			return fmt.Errorf("missing container id or command")
		}
		containerId := ctx.Args().Get(0)
		var cmdArr []string
		cmdArr = append(cmdArr, ctx.Args().Tail()...)
		// 执行命令
		ExecContainerCommand(containerId, cmdArr)
		return nil
	},
}

var stopCommand = cli.Command{
	Name: "stop",
	Usage: "stop a container, eg: ./mydocker stop 容器ID",
	Action: func (ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container id")
		}
		containerId := ctx.Args().Get(0)
		StopContainer(containerId)
		return nil
	},
}