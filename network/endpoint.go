package network

import (
	"mydocker/container"
	"net"

	"github.com/vishvananda/netlink"
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

func configEndpointIPAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	return nil
}

func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	return nil
}
