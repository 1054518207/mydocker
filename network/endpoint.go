package network

import (
	"fmt"
	"mydocker/container"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// 定义网络端点，用于连接容器与网络，保证容器内部与网络的通信
type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"device"`
	IpAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	PortMapping []string         `json:"portmapping"`
	Network     *Network
}

// 配置容器网络端点的地址和路由
func configEndpointIPAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	// 通过网络端点中 "Veth" 的另一端
	l, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}

	// 将容器的网络端点加入到容器的网络空间中，并使这个函数下面的操作都在这个网络空间中进行
	// 执行完函数后，恢复为默认的网络空间
	// 注意此处最后有个 () ， 不加括号不执行defer后的函数
	defer enterContainerNetNS(&l, cinfo)()

	// 获取容器的IP地址及网段，用于配置容器内部接口地址
	// 比如容器 IP 是 192.168.1.2 ， 而网络的网段是 192.168.1.0/24 ，那么这里删除的IP地址字符串就是 192.168.1.2/24 ，用于Veth端点配置
	interfaceIp := *ep.Network.IpRange
	interfaceIp.IP = ep.IpAddress
	// 调用 setinterfaceIP 函数设置容器内 Veth 端点的IP
	if err := setInterfaceIP(ep.Device.PeerName, interfaceIp.String()); err != nil {
		return fmt.Errorf("%s,%v,%s", ep.Device.PeerName, ep.Network, err)
	}

	// 启动容器内的Veth
	if err := setInterfaceUP(ep.Device.PeerName); err != nil{
		return err
	}

	// Net Namespace中默认本地地址127.0.0.1的"lo"网卡默认关闭，需要手动开启
	if err := setInterfaceUP("lo"); err != nil{
		return err
	}

	// 设置容器内的外部请求都是通过容器内的Veth端点访问
	// 0.0.0.0/0网段表示所有的IP地址
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	// 构建需要添加的路由数据，包括网络设备、网关IP以及目的网段
	// 相当于 route add -net 0.0.0.0/0 gw {bridge网桥地址} dev {容器内Veth端点设备}
	defaultRoute := &netlink.Route{
		LinkIndex: l.Attrs().Index,
		Gw: ep.Network.IpRange.IP,
		Dst: cidr,
	}

	// 调用netlink的RouteAdd，添加路由到容器的网络空间
	// RouteAdd函数相当于route add命令
	if err := netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

// 配置端口映射
func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	// 遍历容器端口映射列表
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port mapping format error, %v", pm)
			continue
		}

		// 使用 exec.Command 方法配置iptables的PREROUTING中添加DNAT规则
		// 将宿主机的端口请求转发到容器的地址和端口上
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", portMapping[0],
			ep.IpAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables output: %v", output)
			return err
		}
	}
	return nil
}

func enterContainerNetNS(link *netlink.Link, cinfo *container.ContainerInfo) func() {

	// 找到容器的Net Namespace
	// /proc/[pid]/ns/net 打开这个文件描述符就可以来操作 Net Namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("error get container net namespace, %v", err)
		return func() {}
	}

	// 获取文件描述符
	fd := f.Fd()

	// 锁定当前Go进程执行的线程，如果不锁定则go语言的goroutine可能被调度到别的线程上，不能保证一直在所需要的网络空间中
	// 使用 runtime.LockOSThread时需要先锁定当前程序执行的线程
	runtime.LockOSThread()

	// 修改网络端点Veth的另外一端，将其移动到容器的 Net Namespace 中
	if err := netlink.LinkSetNsFd(*link, int(fd)); err != nil {
		logrus.Errorf("error set link netns, %v", err)
		return func() {
			f.Close()
		}
	}

	// 通过 netns.Get方法获取当前网络的net namespace
	// 目的是方便从容器的Net Namespace中退出，回到原本网络的Net Namespace中
	origins, err := netns.Get()
	if err != nil {
		logrus.Errorf("error get current netns, %v", err)
		return func ()  {
			// 取消对当前程序的线程锁定
			runtime.UnlockOSThread()
			// 关闭Namespace文件
			f.Close()
		}
	}

	// 调用netns.Set方法，将当前进程加入容器的Net Namespace
	if err := netns.Set(netns.NsHandle(fd)); err != nil{
		logrus.Errorf("error set netns, %v", err)
		return func ()  {
			// 关闭Namespace文件
			origins.Close()
			// 取消对当前程序的线程锁定
			runtime.UnlockOSThread()
			// 关闭Namespace文件
			f.Close()
		}
	}

	// 返回之前 Net Namespace 的函数
	// 在容器网络空间中，执行完容器配置之后调用此函数就可以将程序恢复到原生的Net Namespace
	return func() {
		// 恢复到上面获取到的之前的Net Namespace
		netns.Set(origins)
		// 关闭Namespace文件
		origins.Close()
		// 取消对当前程序的线程锁定
		runtime.UnlockOSThread()
		// 关闭Namespace文件
		f.Close()
	}
}