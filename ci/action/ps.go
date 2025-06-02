/*
 * Copyright 2024 The RuleGo Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package action

import (
	"encoding/json"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/utils/maps"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"time"
)

func init() {
	_ = rulego.Registry.Register(&PsNode{})

}

const (
	// OptionsHostInfo 查询主机信息
	OptionsHostInfo = "host/info"
	// OptionsCpuInfo 查询CPU信息
	OptionsCpuInfo = "cpu/info"
	// OptionsCpuPercent 查询CPU使用率
	OptionsCpuPercent = "cpu/percent"
	// OptionsVirtualMemory 查询虚拟内存信息
	OptionsVirtualMemory = "mem/virtualMemory"
	// OptionsSwapMemory 查询交换内存信息
	OptionsSwapMemory = "mem/swapMemory"
	// OptionsDiskUsage 查询磁盘使用情况
	OptionsDiskUsage = "disk/usage"
	// OptionsDiskIOCounters 查询磁盘IO计数器信息
	OptionsDiskIOCounters = "disk/ioCounters"
	// OptionsNetIOCounters 查询网络IO计数器信息
	OptionsNetIOCounters = "net/ioCounters"
	// OptionsInterfaces 查询网络接口信息
	OptionsInterfaces = "net/interfaces"
)

// PsNodeConfiguration 组件配置
type PsNodeConfiguration struct {
	// 指定要查询的指标列表
	// 可选值：
	//  - host/info: 查询主机信息
	//  - cpu/info: 查询CPU信息
	//  - cpu/percent: 查询CPU使用率
	//  - mem/virtualMemory: 查询虚拟内存信息
	//  - mem/swapMemory: 查询交换内存信息
	//  - disk/usage: 查询磁盘使用情况
	//  - disk/ioCounters: 查询磁盘IO计数器信息
	//  - net/ioCounters: 查询网络IO计数器信息
	//  - net/interfaces: 查询网络接口信息
	// 如果为空，则查询所有指标
	Options []string
}

// PsNode 查询主机信息，如：主机信息、CPU信息、内存信息、磁盘信息、网络信息等
type PsNode struct {
	Config PsNodeConfiguration
	// 是否查询所有指标
	All bool
	// 查询指标列表
	Metrics map[string]bool
}

// Type 组件类型
func (x *PsNode) Type() string {
	return "ci/ps"
}

func (x *PsNode) New() types.Node {
	return &PsNode{Config: PsNodeConfiguration{}}
}

// Init 初始化
func (x *PsNode) Init(ruleConfig types.Config, configuration types.Configuration) error {
	err := maps.Map2Struct(configuration, &x.Config)
	x.All = len(x.Config.Options) == 0
	x.Metrics = make(map[string]bool)
	for _, item := range x.Config.Options {
		x.Metrics[item] = true
	}
	return err
}

// OnMsg 处理消息
func (x *PsNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	result := make(map[string]interface{})

	// 查询主机信息
	if x.All || x.contains(OptionsHostInfo) {
		hostInfo, _ := host.Info()
		result[OptionsHostInfo] = hostInfo
	}
	// 查询 CPU 信息
	if x.All || x.contains(OptionsCpuInfo) {
		cpuInfo, _ := cpu.Info()
		result[OptionsCpuInfo] = cpuInfo
	}
	// 查询 CPU 使用率
	if x.All || x.contains(OptionsCpuPercent) {
		percent, _ := cpu.Percent(time.Second, false)
		result[OptionsCpuPercent] = percent
	}

	// 查询虚拟内存信息
	if x.contains(OptionsVirtualMemory) {
		memInfo, _ := mem.VirtualMemory()
		result[OptionsVirtualMemory] = memInfo
	}
	// 查询交换内存信息
	if x.contains(OptionsSwapMemory) {
		swapInfo, _ := mem.SwapMemory()
		result[OptionsSwapMemory] = swapInfo
	}
	// 查询磁盘使用情况
	if x.contains(OptionsDiskUsage) {
		diskInfo, _ := disk.Partitions(true)
		var diskUsages []*disk.UsageStat
		if diskInfo != nil {
			for _, part := range diskInfo {
				diskUsage, _ := disk.Usage(part.Mountpoint)
				if diskUsage != nil {
					diskUsages = append(diskUsages, diskUsage)
				}
			}
		}
		result[OptionsDiskUsage] = diskUsages
	}
	// 查询磁盘IO计数器信息
	if x.contains(OptionsDiskIOCounters) {
		diskIOCounters, _ := disk.IOCounters()
		var items []disk.IOCountersStat
		if diskIOCounters != nil {
			for _, item := range diskIOCounters {
				items = append(items, item)
			}
		}
		result[OptionsDiskIOCounters] = items
	}
	// 查询网络IO计数器信息
	if x.contains(OptionsNetIOCounters) {
		netIOCounters, _ := net.IOCounters(true)
		result[OptionsNetIOCounters] = netIOCounters
	}
	// 查询网络接口信息
	if x.contains(OptionsInterfaces) {
		netInterfaces, _ := net.Interfaces()
		result[OptionsInterfaces] = netInterfaces
	}

	// 将 result 转换为 JSON 字符串并放入 msg.Data
	resultJSON, _ := json.Marshal(result)
	msg.SetData(string(resultJSON))

	ctx.TellSuccess(msg)
}

// 判断是否要查询指定指标
func (x *PsNode) contains(target string) bool {
	if x.All {
		return true
	}
	_, ok := x.Metrics[target]
	return ok
}

// Destroy 销毁
func (x *PsNode) Destroy() {
}
