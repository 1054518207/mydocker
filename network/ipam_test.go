package network

import (
	"net"
	"testing"
)

func TestAllocate(t *testing.T){
	_, ipnet, _ := net.ParseCIDR("192.168.3.0/24")
	ip, _ := ipAllocator.Allocate(ipnet)
	t.Logf("alloc ip %v", ip)
}

func TestRelease(t *testing.T){
	ip, ipnet, _ := net.ParseCIDR("192.168.10.0/24")
	err := ipAllocator.Release(ipnet, &ip)
	if err != nil {
		t.Logf("error %v", err)
	}else{
		t.Logf("OK")
	}
}

func TestInit(t *testing.T) {
	Init()
	t.Logf("%v", networks["testbridge"])
}