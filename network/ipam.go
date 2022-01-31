package network

import (
	"encoding/json"
	"mydocker/util"
	"net"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
)

const ipamDefaultAllocatorPath = "/var/lib/mydocker/network/ipam/subnet.json"

// 存放IP地址分配信息
type IPAM struct {
	// 分配文件存放位置
	SubnetAllocatorPath string
	// 网段和位图算法的数组map，key是网段，value是分配的位图数组
	Subnets *map[string]string
}

var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

func (ipam *IPAM) load() error{
	exist, err := util.FileOrDirExits(ipam.SubnetAllocatorPath)
	if err != nil{
		return err
	}
	if !exist {
		pth, _ := path.Split(ipamDefaultAllocatorPath)
		_ = os.MkdirAll(pth, 0644)
		_, err = os.Create(ipamDefaultAllocatorPath)
		return err
	}
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	if err != nil {
		return err
	}
	defer subnetConfigFile.Close()

	contentBytes := make([]byte, 2000)
	n, err := subnetConfigFile.Read(contentBytes)
	if err != nil{
		return err
	}
	err = json.Unmarshal(contentBytes[:n], ipam.Subnets)
	if err != nil {
		return err
	}
	return nil
}

// 将IPAM信息写入配置文件
func (ipam *IPAM) dump() error {
	exist, err := util.FileOrDirExits(ipam.SubnetAllocatorPath)
	if err != nil{
		return err
	}
	if !exist {
		dir, _ := path.Split(ipam.SubnetAllocatorPath)
		return os.MkdirAll(dir, 0644)
	}
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC | os.O_CREATE | os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer subnetConfigFile.Close()

	ipamSubnet, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}

	_ , err = subnetConfigFile.Write(ipamSubnet)
	if err != nil{
		return err
	}
	return nil
}


func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	// 存放网段中地址分配信息的数组
	ipam.Subnets = &map[string]string{}

	// 获取配置信息
	err = ipam.load()
	if err != nil {
		return nil, err
	}

	_, sub, err := net.ParseCIDR(subnet.String())
	if err != nil {
		return nil, err
	}

	// for subnet: 10.10.0/24, its mask is 255.255.255.0
	// so 'ones' will be 24 and 'bits' will be 32.
	ones, bits := sub.Mask.Size()

	if _, exist := (*ipam.Subnets)[sub.String()]; !exist{
		// 分配位图
		(*ipam.Subnets)[sub.String()] = strings.Repeat("0", 1 << uint8(bits-ones))
	}

	for idx := range (*ipam.Subnets)[sub.String()] {
		if (*ipam.Subnets)[sub.String()][idx] == '0' {
			ip = sub.IP
			ipalloc := []rune((*ipam.Subnets)[sub.String()])
			ipalloc[idx] = '1'
			(*ipam.Subnets)[sub.String()] = string(ipalloc)
			for t := 3; t >= 0; t-- {
				// TODO: 这种写法能导致溢出错误，不知道原书为何这么写，总之需要 idx >> x后结果在0-255之间，否则会出现溢出bug
				ip[3-t] += uint8(idx >> (t*8))
			}
			// TODO: 这种写法有bug，如果在最后一部分IP为255，+1得256导致错误
			ip[3] += 1
			break
		}
	}
	
	ipam.dump()

	return 
}

func (ipam *IPAM) Release(subnet *net.IPNet, ipAddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	err := ipam.load()
	if err != nil {
		logrus.Errorf("Error dump allocation info, error %v", err)
		return err
	}

	// 计算IP地址在网段位图数组中得索引位置
	idx := 0
	// 将IP地址转换为4个字节的表示方式
	releaseIP := ipAddr.To4()
	// TODO: 由于IP是从1开始分配的，所以转换成索引应减1,仅限在子网掩码在24时奏效
	releaseIP[3] -= 1
	for t := uint(4) ; t > 0 ; t--{
		idx += int(releaseIP[t-1] - subnet.IP.To4()[t-1]) << ((4-t) * 8)
		logrus.Info(int(releaseIP[t-1] - subnet.IP[t-1]) << ((4-t) * 8))
	}
	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[idx] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	// 保存释放掉IP后的网段IP分配信息
	ipam.dump()
	return nil
}
