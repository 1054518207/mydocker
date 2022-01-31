package network

import (
	"encoding/json"
	"fmt"
	"mydocker/container"
	"mydocker/util"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
)

type Network struct {
	Name    string     // 网络名
	IpRange *net.IPNet // IP 地址段
	Driver  string     // 网络驱动名称
}

type NetworkDriver interface {
	// 驱动名称
	Name() string
	// 创建网络
	Create(subnet string, name string) (*Network, error)
	// 删除网络
	Delete(network Network) error
	// 连接容器网络端点到网络
	Connect(network *Network, endpoint *Endpoint) error
	// 从网络上移除容器网络端点
	Disconnect(network *Network, endpoint *Endpoint) error
}

var (
	drivers            map[string]NetworkDriver
	networks           map[string]*Network
	defaultNetworkPath = "/var/lib/mydocker/network/network/"
)

func (nw *Network) dump(dumpPath string) error {
	// 检查目录是否存在，不存在则创建目标目录
	exist, err := util.FileOrDirExits(dumpPath)
	if err != nil {
		return fmt.Errorf("can not detect network dump path %s error %v", dumpPath, err)
	}
	if !exist {
		os.MkdirAll(dumpPath, os.ModePerm)
	}
	// 保存文件名称为网络名
	nwPath := path.Join(dumpPath, nw.Name)

	// 打开保存文件用于写入，模式参数为存在内容则清空，只写入，不存在则创建
	// O_TRUNC 如果文件已存在，打开时将会清空文件内容
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("open file %s error %v", nwPath, err)
		return err
	}
	defer nwFile.Close()

	// 通过Json库序列化网络对象到json字符串
	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("error marshal network object json error %v", err)
		return err
	}

	// 网络配置的json字符串写入文件
	if _, err := nwFile.Write(nwJson); err != nil {
		logrus.Errorf("error write network object json error %v", err)
		return err
	}
	return nil
}

func (nw *Network) load(dumpPath string) error {
	// 检查目录是否存在，不存在则创建目标目录
	exist, err := util.FileOrDirExits(dumpPath)
	if err != nil {
		return fmt.Errorf("can not detect network dump path %s error %v", dumpPath, err)
	}
	if !exist {
		return fmt.Errorf("can not find network dump path %s error %v", dumpPath, err)
	}
	nwConfigFile, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	// 从配置文件中读取网络配置json字符串
	contentBytes := make([]byte, 2000)
	n, err := nwConfigFile.Read(contentBytes)
	if err != nil {
		return err
	}
	err = json.Unmarshal(contentBytes[:n], nw)
	if err != nil {
		logrus.Errorf("error unmarshal %s network json data error %v", dumpPath, err)
		return err
	}
	return nil
}

func (nw *Network) remove(dumpPath string) error {
	exist, err := util.FileOrDirExits(dumpPath)
	if err != nil {
		return fmt.Errorf("can not judge network dir %s error %v", dumpPath, err)
	}
	if !exist {
		return nil
	}
	return os.Remove(path.Join(dumpPath, nw.Name))
}

// 创建网络
func CreateNetwork(driver, subnet, name string) error {
	// ParseCIDR将子网段字符串转化为 net.IPNet 对象
	_, cidr, _ := net.ParseCIDR(subnet)
	// 通过IPAM分配网关IP，获取网段中第一个ip作为网关ip
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIp

	//调用指定的网络驱动创建网络，此处的drivers字典是各个网络驱动的实例字典，通过调用网络驱动的Create方法创建网络
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	nw.Driver = driver

	// 保存网络信息，将网络信息保存在文件系统中，方便查询和在网络上连接端点
	return nw.dump(defaultNetworkPath)
}

func Connect(networkName string, cinfo *container.ContainerInfo) error {
	// 从networks字典中渠道容器连接的网络信息，networks字典中保存了当前已经创建的网络
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network: %s", networkName)
	}
	// 通过IPAM从网络的网段中获取可用的IP作为容器IP地址
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}
	// 创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.Id, networkName),
		IpAddress:   ip,
		Network:     network,
		PortMapping: cinfo.PortMapping,
	}
	
	logrus.Infof("network.go, Connet: ID = %s; IP: %s; Network: %s", ep.ID, ep.IpAddress, ep.Network.IpRange)
	
	// 调用网络驱动Connet方法挂载和配置网络端点
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}

	// 进入到容器的网络Nampespace配置容器网络设备的IP地址和路由
	if err = configEndpointIPAddressAndRoute(ep, cinfo); err != nil {
		return err
	}
	
	// 配置容器到宿主机的端口映射
	return configPortMapping(ep, cinfo)
}

// https://github.com/xianlubird/mydocker/issues/52
// 用于 ./mydocker network list 命令查询当前创建的网络
func Init() error {

	// need to reset the rule of iptables FORWARD chain to ACCEPT, because
	// docker 1.13+ changed the default iptables forwarding policy to DROP
	// https://github.com/moby/moby/pull/28257/files
	// https://github.com/kubernetes/kubernetes/issues/40182
	// 看下你的环境iptables FORWARD链默认策略是什么，用这个查看iptables-save -t filter; 如果是DROP的话，试下这个iptables -P FORWARD ACCEPT
	enableForwardCmd := exec.Command("iptables", "-P", "FORWARD", "ACCEPT")
	if err := enableForwardCmd.Run(); err != nil {
		return fmt.Errorf("failed to execute cmd %s", enableForwardCmd.Args)
	}


	// 判断网络配置目录是否存在，不存在则创建
	exist, err := util.FileOrDirExits(defaultNetworkPath)
	if err != nil {
		return nil
	}
	if !exist {
		os.MkdirAll(defaultNetworkPath, os.ModePerm)
	}

	// 初始化设备
	drivers = make(map[string]NetworkDriver)
	networks = make(map[string]*Network)

	// 加载网络驱动
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	// 检查网络配置目录中的所有文件
	// filepath.Walk(path, func(string, os.FileInfo, error)) 函数会便利指定path目录，并执行第二个参数中函数指针处理每一个文件
	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		// 目录跳过
		if info.IsDir() {
			return nil
		}

		// 加载文件名作为网络名
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}

		// 调用load方法加载网络配置信息
		if err := nw.load(nwPath); err != nil {
			logrus.Errorf("error load network: %s", err)
		}

		// 将网络配置信息加入到 networks 字典中
		networks[nwName] = nw
		return nil
	})

	return nil
}

func ListNetwork() {
	// 使用tabwriter进行网络展示
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	// 遍历网络信息
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n", nw.Name, nw.IpRange, nw.Driver)
	}
	// 输出到标准输出
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	// 查找目标网络是否存在
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network: %s", networkName)
	}
	logrus.Debugf("Delete network info load, Driver: %s; Name: %s; IPRange: %s", nw.Driver, nw.Name, nw.IpRange)

	// 调用 IPAM 的实例 ipAllocator 释放网络的网关IP
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return fmt.Errorf("error remove network gateway IP : %s", err)
	}

	// 调用网络驱动删除网络创建的设备与配置
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("error remove network driver error: %s", err)
	}
	return nw.remove(defaultNetworkPath)
}
