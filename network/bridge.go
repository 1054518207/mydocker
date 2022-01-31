package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type BridgeNetworkDriver struct{}

func (b *BridgeNetworkDriver) Name() string{
	return "bridge"
}

func (b *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error){
	// 通过net包中的net.ParseCIDR方法获取网段字符串的网关IP地址和网络IP段
	ip, ipNet, _ := net.ParseCIDR(subnet)
	ipNet.IP = ip
	// 初始化网络对象
	nw := &Network{
		Name: name,
		IpRange: ipNet,
	}
	// 配置Linux Bridge
	err := b.initBridge(nw)
	if err != nil {
		logrus.Errorf("error init bridge")
	}
	// 返回配置好的网络
	return nw, err
}

// 删除bridge网络,相当于 ip link delete bridgeName type bridge
func (b *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	l, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(l)
}

// 连接容器网络端点到网络
func (b *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	return nil
}

// 从网络上移除容器网络端点
func (b *BridgeNetworkDriver) Disconnect(network *Network, endpoint *Endpoint) error {
	return nil
}

func (b *BridgeNetworkDriver) initBridge(nw *Network) error {
	// 创建Bridge虚拟设备
	bridgeName := nw.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return err
	}
	logrus.Infof("init bridge, target bridge name %s", bridgeName)

	// 设置Bridge设备的地址和路由
	gatewayIP := nw.IpRange
	gatewayIP.IP = nw.IpRange.IP
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("Error assigning address: %s on bridge: %s with an error of: %v", gatewayIP, bridgeName, err)
	}
	
	// 启动Bridge设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("Error set bridge up: %s, Error: %v", bridgeName, err)
	}


	// 设置iptables的SNAT规则
	if err := setIpTables(bridgeName, nw.IpRange); err != nil {
		return fmt.Errorf("Error setting iptables for %s: %v", bridgeName, err)
	}
	
	return nil
}

func createBridgeInterface(bridgeName string) error{
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error() ,"no such network interface"){
		return fmt.Errorf("%s interface is existed", bridgeName)
	}
	
	// 初始化netlink对象，并添加link属性
	// https://github.com/vishvananda/netlink
	la := netlink.NewLinkAttrs()
    la.Name = bridgeName
	mybridge := &netlink.Bridge{LinkAttrs: la}
	err = netlink.LinkAdd(mybridge)
    if err != nil  {
        return fmt.Errorf("could not add %s: %v", la.Name, err)
    }
	return nil
}

func setInterfaceIP(bridgeName, ip string) error {
	iface, err := netlink.LinkByName(bridgeName)
	if err != nil{
		return err
	}
	ipNet, err := netlink.ParseIPNet(ip)
	if err != nil {
		logrus.Errorf("error parse IPNet %s", ip)
		return err
	}
	addr := netlink.Addr{
		IPNet: ipNet,
		Label: "",
		Flags: 0,
		Scope: 0,
	}

	return netlink.AddrAdd(iface, &addr)
}

func setInterfaceUP(interfaceName string) error {
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("Error retrieving a link named [ %s ]: %v", iface.Attrs().Name, err)
	}

	// 通过netlink的 "LinkSetUp" 方法设置接口为 "UP" 状态
	// 等价于 ip link set xxx up
	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("Error enabling interface for %s: %v", interfaceName, err)
	}
	return nil
}

// 设置 iptables 对应的bridge的 MASQUERADE 规则
func setIpTables(bridgeName string, subnet *net.IPNet) error {
	// 使用命令行的方式添加iptables规则
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	// 执行iptables命令配置SNAT规则
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("iptables set failed, %v", err)
	}
	logrus.Infof("iptables outputs: %s", output)
	return nil
}