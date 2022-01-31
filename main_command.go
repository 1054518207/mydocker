package main

import (
	"fmt"
	"mydocker/cgroups/subsystems"
	"mydocker/container"
	"mydocker/images"
	"mydocker/network"
	"mydocker/util"
	"os"
	"path"

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
		cli.StringSliceFlag{Name: "e", Usage: "set environment"},
		cli.StringFlag{Name: "net", Usage: "container network"},
		cli.StringSliceFlag{ Name: "p", Usage: "port mapping"},
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
		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]
		imagePath := path.Join(images.ImagesStoreDir, imageName)
		exist, _ := util.FileOrDirExits(imagePath)
		if !exist{
			return fmt.Errorf("image name %s not found in %s", imageName, images.ImagesStoreDir)
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
		nw := context.String("net")

		envs := context.StringSlice("e")
		portmapping := context.StringSlice("p")

		Run(tty, cmdArray, resConf, volume, containerName, imageName, envs, nw, portmapping)
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
	Usage: "commit a container into image, eg: ./mydocker commit 9871200000. This will be a tar in /root dir.",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerId := ctx.Args().Get(0)
		commitContainer(containerId)
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

var rmCommand = cli.Command{
	Name: "rm",
	Usage: "remove a stopped container, eg: ./mydocker rm 容器ID",
	Action: func (ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container id")
		}
		containerId := ctx.Args().Get(0)
		RemoveContainer(containerId)
		return nil
	},
}

var networkCommand = cli.Command{
	Name: "network",
	Usage: "container network commands",
	Subcommands: []cli.Command{
		{
			Name: "create",
			Usage: "create a container network, eg: ./mydocker network create --driver bridge --subnet 192.168.10.1/24 mybridge",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "driver", Usage: "network driver"},
				cli.StringFlag{Name: "subnet", Usage: "subnet CIDR, eg: 192.168.10.0/24"},
			},
			Action: func (ctx *cli.Context) error {
				if len(ctx.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}
				network.Init()
				if err := network.CreateNetwork(ctx.String("driver"), ctx.String("subnet"), ctx.Args()[0]); err != nil {
					return fmt.Errorf("create network error: %+v", err)
				}
				return nil
			},
		},
		{
			Name: "list",
			Usage: "list container network",
			Action: func (ctx *cli.Context)  {
				network.Init()
				network.ListNetwork()
			},
		},
		{
			Name: "remove",
			Usage: "remove container network, eg: mydocker network remove [network name]",
			Action: func (ctx *cli.Context) error {
				if len(ctx.Args()) < 1{
					return fmt.Errorf("missing network name")
				}else if len(ctx.Args()) > 1 {
					return fmt.Errorf("network name is one word")
				}
				network.Init()
				// 此部分需要注意，删除网桥时，NAT的POSTROUTING部分还没有删除，如果需要删除需要手动
				// iptables -t nat -vnL POSTROUTING --line-numbers   此命令查看 POSTROUTING 部分，num表示行号
				// iptables -t nat -D POSTROUTING {行号} 删除指定行号POSTROUTING
				err := network.DeleteNetwork(ctx.Args()[0])
				if err != nil{
					return fmt.Errorf("remove network error: %+v", err)
				}
				return nil
			},
		},
	},
}