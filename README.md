# cSphere autoscale controller
每秒检查一下指定服务下所有容器的平均CPU利用率和内存占用量。
如果连续5次检查结果都超出上限，增加1个容器。

如果连续5次检查结果都低于下限，减少一个容器。

用户可以配置最大容器数量和最少容器数量。

Docker 镜像：`docker pull index.csphere.cn/csphere/autoscale`


**注意：**由于OEM版本cSphere后端删除了`addcontainer`、`delcontainer`这两个API，新增了`changesum`API，所以OEM版本请使用`index.csphere.cn/csphere/autoscale:new`这个镜像，否则无法扩容。

## 编译方法
本项目利用go 1.6引入的vendor功能管理依赖包。
请使用go 1.6及以上版本进行编译，编译方法为在项目根目录下执行`make`。

如果使用go 1.5，可以设置`GO15VENDOREXPERIMENT=1`环境变量然后编译。

低版本的go需要自行处理依赖。

## 使用方法
配置文件路径： `/etc/autoscale.json`

配置文件格式参考[sample-config.json](sample-config.json).

```json
{
  "ApiKey": "fbfed031cadbfa4b661c9cf0916ed5ce78637038",
  "ControllerAddr": "http://192.168.122.110/",
  "Groups": [
    {
    	"App": "myapp",
    	"Service": "api",
    	"CpuLow": 5,
    	"CpuHigh": 10,
    	"MemoryLow": "15m",
    	"MemoryHigh": "20m",
    	"MaxContainers": 2,
    	"MinContainers": 1
    }
  ]
}
```

各字段说明：

- ApiKey: cSphere控制器的ApiKey，可以在cSphere管理面板的“设置”页面创建
- ControllerAddr: cSphere控制器的地址，格式为：http://controller-host:port/
- Groups: 这个数组里配置所有需要启用自动伸缩功能的服务列表
- App: 应用名称
- Service: 服务名称
- CpuLow: 服务中各容器的平均CPU利用率低于`CpuLow`**且**平均内存消耗低于`MemoryLow`时，容器数量自动减少1个，容器最低数量由`MinContainers`决定
- CpuHigh: 服务中各容器的平均CPU利用率大于这个值时，容器数量自动加1, 容器数量最大值由`MaxContainers`决定
- MemoryHigh: 服务中各容器的平均内存消耗大于这个值时，容器数量自动加1, 容器数量最大值由`MaxContainers`决定

