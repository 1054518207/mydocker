package subsystems

// ResourceConfig 用于传递资源限制配置的结构体
type ResourceConfig struct {
	MemoryLimit string // 内存限制
	CpuShare    string // CPU时间片权重
	CpuSet      string // CPU核心数
}

// Subsystem 接口，每个Subsystem可以实现下面四个接口
// 此处将cgroup抽象成path，原因是cgroup在hierarchy的路径是虚拟文件系统中的虚拟路径
type Subsystem interface {
	Name() string                               // 返回 subsystem 的名字，例如： cpu memory
	Set(path string, res *ResourceConfig) error // 设置某个cgroup在此subsystem中的资源限制
	Apply(path string, pid int) error           // 将进程添加到某个 cgroup 中
	Remove(path string) error                   // 移除某个 cgroup
}

var SubsystemsIns = []Subsystem{
	&CpusetSubSystem{},
	&MemorySubSystem{},
	&CpuSubSystem{},
}
