package models

import (
	harper "github.com/HarperDB/sdk-go"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"slices"
	"strconv"
	"time"
)

func systemToFields(sysInfo *harper.SysInfo) data.Fields {
	return data.Fields{
		data.NewField("System.Platform", nil, []string{sysInfo.System.Platform}),
		data.NewField("System.Distro", nil, []string{sysInfo.System.Distro}),
		data.NewField("System.Release", nil, []string{sysInfo.System.Release}),
		data.NewField("System.Codename", nil, []string{sysInfo.System.Codename}),
		data.NewField("System.Kernel", nil, []string{sysInfo.System.Kernel}),
		data.NewField("System.Arch", nil, []string{sysInfo.System.Arch}),
		data.NewField("System.Hostname", nil, []string{sysInfo.System.Hostname}),
		data.NewField("System.NodeVersion", nil, []string{sysInfo.System.NodeVersion}),
		data.NewField("System.NPMVersion", nil, []string{sysInfo.System.NPMVersion}),
	}
}

func timeToFields(sysInfo *harper.SysInfo) data.Fields {
	return data.Fields{
		data.NewField("Time.Current", nil, []time.Time{sysInfo.Time.Current.ToTime()}),
		data.NewField("Time.Uptime", nil, []float64{sysInfo.Time.Uptime}),
		data.NewField("Time.Timezone", nil, []string{sysInfo.Time.Timezone}),
		data.NewField("Time.TimezoneName", nil, []string{sysInfo.Time.TimezoneName}),
	}
}

//func mapSlice[T, V any](slice []T, fn func(T) V) []V {
//	result := make([]V, len(slice))
//	for i, v := range slice {
//		result[i] = fn(v)
//	}
//	return result
//}

//func joinFloats(floats []float64) string {
//	return strings.Join(mapSlice(floats, func(f float64) string {
//		return strconv.FormatFloat(f, 'f', -1, 64)
//	}), ",")
//}

func cpuToFields(sysInfo *harper.SysInfo) data.Fields {
	fields := data.Fields{
		data.NewField("CPU.Manufacturer", nil, []string{sysInfo.CPU.Manufacturer}),
		data.NewField("CPU.Brand", nil, []string{sysInfo.CPU.Brand}),
		data.NewField("CPU.Vendor", nil, []string{sysInfo.CPU.Vendor}),
		data.NewField("CPU.Speed", nil, []float64{sysInfo.CPU.Speed}),
		data.NewField("CPU.Cores", nil, []int32{int32(sysInfo.CPU.Cores)}),
		data.NewField("CPU.PhysicalCores", nil, []int32{int32(sysInfo.CPU.PhysicalCores)}),
		data.NewField("CPU.Processors", nil, []int32{int32(sysInfo.CPU.Processors)}),

		data.NewField("CPU.CPUSpeed.Min", nil, []float64{sysInfo.CPU.CPUSpeed.Min}),
		data.NewField("CPU.CPUSpeed.Max", nil, []float64{sysInfo.CPU.CPUSpeed.Max}),
		data.NewField("CPU.CPUSpeed.Avg", nil, []float64{sysInfo.CPU.CPUSpeed.Avg}),

		data.NewField("CPU.CurrentLoad.AvgLoad", nil, []float64{sysInfo.CPU.CurrentLoad.AvgLoad}),
		data.NewField("CPU.CurrentLoad.CurrentLoad", nil, []float64{sysInfo.CPU.CurrentLoad.CurrentLoad}),
		data.NewField("CPU.CurrentLoad.CurrentLoadUser", nil, []float64{sysInfo.CPU.CurrentLoad.CurrentLoadUser}),
		data.NewField("CPU.CurrentLoad.CurrentLoadSystem", nil, []float64{sysInfo.CPU.CurrentLoad.CurrentLoadSystem}),
		data.NewField("CPU.CurrentLoad.CurrentLoadNice", nil, []float64{sysInfo.CPU.CurrentLoad.CurrentLoadNice}),
		data.NewField("CPU.CurrentLoad.CurrentLoadIdle", nil, []float64{sysInfo.CPU.CurrentLoad.CurrentLoadIdle}),
		data.NewField("CPU.CurrentLoad.CurrentLoadIRQ", nil, []float64{sysInfo.CPU.CurrentLoad.CurrentLoadIRQ}),
	}

	for idx, cpu := range sysInfo.CPU.CPUs {
		// Leaving CPU.CPUSpeed.Cores out for now; let's see if anyone actually uses it

		labels := data.Labels{"CPU": strconv.Itoa(idx)}
		fields = append(fields, data.Fields{
			data.NewField("CPU.CPUs.Load", labels, []float64{cpu.Load}),
			data.NewField("CPU.CPUs.LoadUser", labels, []float64{cpu.LoadUser}),
			data.NewField("CPU.CPUs.LoadSystem", labels, []float64{cpu.LoadSystem}),
			data.NewField("CPU.CPUs.LoadNice", labels, []float64{cpu.LoadNice}),
			data.NewField("CPU.CPUs.LoadIdle", labels, []float64{cpu.LoadIdle}),
			data.NewField("CPU.CPUs.LoadIRQ", labels, []float64{cpu.LoadIRQ}),
		}...)
	}

	return fields
}

func memoryToFields(sysInfo *harper.SysInfo) data.Fields {
	return data.Fields{
		data.NewField("Memory.Total", nil, []int64{sysInfo.Memory.Total}),
		data.NewField("Memory.Free", nil, []int64{sysInfo.Memory.Total}),
		data.NewField("Memory.Used", nil, []int64{sysInfo.Memory.Used}),
		data.NewField("Memory.Active", nil, []int64{sysInfo.Memory.Active}),
		data.NewField("Memory.Available", nil, []int64{sysInfo.Memory.Available}),
		data.NewField("Memory.SwapTotal", nil, []int64{sysInfo.Memory.SwapTotal}),
		data.NewField("Memory.SwapUsed", nil, []int64{sysInfo.Memory.SwapUsed}),
		data.NewField("Memory.SwapFree", nil, []int64{sysInfo.Memory.SwapFree}),
	}
}

func diskToFields(sysInfo *harper.SysInfo) data.Fields {
	fields := data.Fields{
		data.NewField("Disk.IO.RIO", nil, []int64{sysInfo.Disk.IO.RIO}),
		data.NewField("Disk.IO.WIO", nil, []int64{sysInfo.Disk.IO.WIO}),
		data.NewField("Disk.IO.TIO", nil, []int64{sysInfo.Disk.IO.TIO}),

		data.NewField("Disk.ReadWrite.RX", nil, []int64{sysInfo.Disk.ReadWrite.RX}),
		data.NewField("Disk.ReadWrite.WX", nil, []int64{sysInfo.Disk.ReadWrite.WX}),
		data.NewField("Disk.ReadWrite.TX", nil, []int64{sysInfo.Disk.ReadWrite.TX}),
		data.NewField("Disk.ReadWrite.MS", nil, []int64{sysInfo.Disk.ReadWrite.MS}),
	}

	for idx, disk := range sysInfo.Disk.Size {
		labels := data.Labels{"Disk": strconv.Itoa(idx)}
		fields = append(fields, data.Fields{
			data.NewField("Disk.Size.FS", labels, []string{disk.FS}),
			data.NewField("Disk.Size.Type", labels, []string{disk.Type}),
			data.NewField("Disk.Size.Size", labels, []int64{disk.Size}),
			data.NewField("Disk.Size.Used", labels, []int64{disk.Used}),
			data.NewField("Disk.Size.Use", labels, []float64{disk.Use}),
			data.NewField("Disk.Size.Mount", labels, []string{disk.Mount}),
		}...)
	}

	return fields
}

func networkInterfacesToFields(sysInfo *harper.SysInfo) data.Fields {
	fields := make(data.Fields, 0)

	for _, iface := range sysInfo.Network.Interfaces {
		labels := data.Labels{"Iface": iface.Iface}
		fields = append(fields, data.Fields{
			data.NewField("Network.Interfaces.Iface", labels, []string{iface.Iface}),
			data.NewField("Network.Interfaces.IfaceName", labels, []string{iface.IfaceName}),
			data.NewField("Network.Interfaces.IP4", labels, []string{iface.IP4}),
			data.NewField("Network.Interfaces.IP6", labels, []string{iface.IP6}),
			data.NewField("Network.Interfaces.Mac", labels, []string{iface.Mac}),
			data.NewField("Network.Interfaces.OperState", labels, []string{iface.OperState}),
			data.NewField("Network.Interfaces.Type", labels, []string{iface.Type}),
			data.NewField("Network.Interfaces.Duplex", labels, []string{iface.Duplex}),
			data.NewField("Network.Interfaces.Speed", labels, []float64{iface.Speed}),
			data.NewField("Network.Interfaces.CarrierChanges", labels, []int64{iface.CarrierChanges}),
		}...)
	}

	return fields
}

func networkStatsToFields(sysInfo *harper.SysInfo) data.Fields {
	fields := make(data.Fields, 0)

	for _, stats := range sysInfo.Network.Stats {
		labels := data.Labels{"Iface": stats.Iface}
		fields = append(fields, data.Fields{
			data.NewField("Network.Stats.Iface", labels, []string{stats.Iface}),
			data.NewField("Network.Stats.OperState", labels, []string{stats.OperState}),
			data.NewField("Network.Stats.RxBytes", labels, []int64{stats.RxBytes}),
			data.NewField("Network.Stats.RxDropped", labels, []int64{stats.RxDropped}),
			data.NewField("Network.Stats.RxErrors", labels, []int64{stats.RxErrors}),
			data.NewField("Network.Stats.TxBytes", labels, []int64{stats.TxBytes}),
			data.NewField("Network.Stats.TxDropped", labels, []int64{stats.TxDropped}),
			data.NewField("Network.Stats.TxErrors", labels, []int64{stats.TxErrors}),
		}...)
	}

	return fields
}

func networkConnectionsToFields(sysInfo *harper.SysInfo) data.Fields {
	fields := make(data.Fields, 0)

	for idx, conn := range sysInfo.Network.Connections {
		labels := data.Labels{"Connection": strconv.Itoa(idx)}
		fields = append(fields, data.Fields{
			data.NewField("Network.Connections.Protocol", labels, []string{conn.Protocol}),
			data.NewField("Network.Connections.LocalAddress", labels, []string{conn.LocalAddress}),
			data.NewField("Network.Connections.LocalPort", labels, []string{conn.LocalPort}),
			data.NewField("Network.Connections.PeerAddress", labels, []string{conn.PeerAddress}),
			data.NewField("Network.Connections.PeerPort", labels, []string{conn.PeerPort}),
			data.NewField("Network.Connections.State", labels, []string{conn.State}),
			data.NewField("Network.Connections.PID", labels, []int64{conn.PID}),
			data.NewField("Network.Connections.Process", labels, []string{conn.Process}),
		}...)
	}

	return fields
}

func networkToFields(sysInfo *harper.SysInfo) data.Fields {
	fields := data.Fields{
		data.NewField("Network.DefaultInterface", nil, []string{sysInfo.Network.DefaultInterface}),
		data.NewField("Network.Latency.URL", nil, []string{sysInfo.Network.Latency.URL}),
		data.NewField("Network.Latency.Ok", nil, []bool{sysInfo.Network.Latency.Ok}),
		data.NewField("Network.Latency.Status", nil, []int64{sysInfo.Network.Latency.Status}),
		data.NewField("Network.Latency.MS", nil, []int64{sysInfo.Network.Latency.MS}),
	}

	fields = append(fields, networkInterfacesToFields(sysInfo)...)
	fields = append(fields, networkStatsToFields(sysInfo)...)
	fields = append(fields, networkConnectionsToFields(sysInfo)...)

	return fields
}

func threadsToFields(sysInfo *harper.SysInfo) data.Fields {
	fields := make(data.Fields, 0)

	for _, thread := range sysInfo.Threads {
		labels := data.Labels{"ThreadID": strconv.FormatInt(thread.ThreadID, 10)}
		fields = append(fields, data.Fields{
			data.NewField("Thread.Name", labels, []string{thread.Name}),
			data.NewField("Thread.HeapTotal", labels, []int64{thread.HeapTotal}),
			data.NewField("Thread.HeapUsed", labels, []int64{thread.HeapUsed}),
			data.NewField("Thread.ExternalMemory", labels, []int64{thread.ExternalMemory}),
			data.NewField("Thread.ArrayBuffers", labels, []int64{thread.ArrayBuffers}),
			data.NewField("Thread.SinceLastUpdate", labels, []int64{thread.SinceLastUpdate}),
			data.NewField("Thread.Idle", labels, []float64{thread.Idle}),
			data.NewField("Thread.Active", labels, []float64{thread.Active}),
			data.NewField("Thread.Utilization", labels, []float64{thread.Utilization}),
		}...)
	}

	return fields
}

func SysInfoToFields(sysInfo *harper.SysInfo, attrs []string) data.Fields {
	fields := make(data.Fields, 0)

	log.DefaultLogger.Debug("Requesting Harper system information attributes", "attrs", attrs)

	if len(attrs) == 0 || slices.Contains(attrs, "system") {
		fields = append(fields, systemToFields(sysInfo)...)
	}

	if len(attrs) == 0 || slices.Contains(attrs, "time") {
		fields = append(fields, timeToFields(sysInfo)...)
	}

	if len(attrs) == 0 || slices.Contains(attrs, "cpu") {
		fields = append(fields, cpuToFields(sysInfo)...)
	}

	if len(attrs) == 0 || slices.Contains(attrs, "memory") {
		fields = append(fields, memoryToFields(sysInfo)...)
	}

	if len(attrs) == 0 || slices.Contains(attrs, "disk") {
		fields = append(fields, diskToFields(sysInfo)...)
	}

	if len(attrs) == 0 || slices.Contains(attrs, "network") {
		fields = append(fields, networkToFields(sysInfo)...)
	}

	if len(attrs) == 0 || slices.Contains(attrs, "threads") {
		fields = append(fields, threadsToFields(sysInfo)...)
	}

	// TOOD: Lots of multi-value data in these two; convert to label differentiation
	//fields = append(fields, harperDBProcessesToFields(sysInfo)...)
	//fields = append(fields, tableSizeToFields(sysInfo)...)

	return fields
}
