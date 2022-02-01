# mydocker

按照书本部分编写代码，参考：https://github.com/xianlubird/mydocker 

修改书中无法运行部分，后续如果有时间则进行完善，此部分还有一些问题，比如使用老旧的AUFS挂载驱动，应该添加overlay2驱动，还有ip地址分配部分，学习作者在书后提供的`etcd`等分布式KV数据库等。后续加油💪
- 注意事项：需要自行使用docker pull一个busybox，然后放置到 `/var/lib/mydocker/images` 目录下，然后运行此程序，目录结构如下所示：
```
.
└── busybox
    ├── bin
    ...
```
- 使用`busybox`提示：
```bash
$ docker pull busybox 
$ docker run -d busybox top
$ docker export -o busybox.tar 此处填写容器id
$ mkdir -p /var/lib/mydocker/images/busybox
$ tar -xvf busybox.tar -C /var/lib/mydocker/images/busybox
```