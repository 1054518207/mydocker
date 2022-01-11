package subsystems

import (
	"log"
	"testing"
)

func TestFindCgroupMountpoint(t *testing.T) {
	pth := FindCgroupMountpoint("memory")
	log.Printf("Mount path: %v", pth)
}

func TestGetCgroupPath(t *testing.T) {
	path, err := GetCgroupPath("memory", "user.slice")
	if err != nil {
		return
	}
	log.Println("get cgroup path:", path)
}
