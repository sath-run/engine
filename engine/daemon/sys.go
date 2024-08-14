package daemon

import (
	"encoding/xml"
	"os/exec"
	"strconv"
	"strings"

	pb "github.com/sath-run/engine/engine/daemon/protobuf"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func GetSystemInfo() *pb.SystemInfo {
	info := pb.SystemInfo{
		Host:   &pb.HostInfo{},
		Cpu:    &pb.CpuInfo{},
		Memory: &pb.MemoryInfo{},
		Gpu:    &pb.GpuInfo{},
	}
	cpus, err := cpu.Info()
	if err != nil {
		info.Cpu.Err = err.Error()
	} else {
		for _, cpu := range cpus {
			cpuInfo := &pb.Cpu{
				Id:        cpu.CPU,
				CacheSize: cpu.CacheSize,
				Clock:     uint64(cpu.Mhz * 1e6),
				ModelName: cpu.ModelName,
			}
			info.Cpu.Cpus = append(info.Cpu.Cpus, cpuInfo)
		}
	}

	hostInfo, err := host.Info()
	if err != nil {
		info.Host.Err = err.Error()
	} else {
		info.Host.PlatformFamily = hostInfo.PlatformFamily
		info.Host.PlatformVersion = hostInfo.PlatformVersion
		info.Host.KernelVersion = hostInfo.KernelVersion
		info.Host.KernelArch = hostInfo.KernelArch
		info.Host.Os = hostInfo.OS
		info.Host.Platform = hostInfo.Platform
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		info.Memory.Err = err.Error()
	} else {
		info.Memory.Total = memInfo.Total
	}

	gpuInfo, err := GetNvidiaGPUInfo()
	if err != nil {
		info.Gpu.Err = err.Error()
	} else {
		info.Gpu.CudaVersion = gpuInfo.CudaVersion
		info.Gpu.DriverVersion = gpuInfo.DriverVersion
		for _, gpu := range gpuInfo.Gpus {
			g := &pb.Gpu{
				Id:                  gpu.Id,
				Uuid:                gpu.Uuid,
				ProductName:         gpu.ProductName,
				ProductBrand:        gpu.ProductBrand,
				ProductArchitecture: gpu.ProductArchitecture,
				VbiosVersion:        gpu.VbiosVersion,
				GpuPartNumber:       gpu.GpuPartNumber,
				Clocks:              &pb.GpuClocks{},
				MaxClocks:           &pb.GpuClocks{},
			}
			g.Clocks.Graphics = parseClock(gpu.Clocks.Graphics)
			g.Clocks.Mem = parseClock(gpu.Clocks.Mem)
			g.Clocks.Sm = parseClock(gpu.Clocks.Sm)
			g.Clocks.Video = parseClock(gpu.Clocks.Video)
			g.MaxClocks.Graphics = parseClock(gpu.MaxClocks.Graphics)
			g.MaxClocks.Mem = parseClock(gpu.MaxClocks.Mem)
			g.MaxClocks.Sm = parseClock(gpu.MaxClocks.Sm)
			g.MaxClocks.Video = parseClock(gpu.MaxClocks.Video)
			info.Gpu.Gpus = append(info.Gpu.Gpus, g)
		}
	}

	return &info
}

func parseClock(clock string) uint64 {
	retval, _ := strconv.ParseUint(strings.TrimSuffix(clock, " MHz"), 10, 64)
	return retval * 1e6
}

type GPUInfo struct {
	Timestamp     string `xml:"timestamp"`
	DriverVersion string `xml:"driver_version"`
	CudaVersion   string `xml:"cuda_version"`
	Gpus          []Gpu  `xml:"gpu"`
}

type Gpu struct {
	Id                  string   `xml:"id,attr"`
	ProductName         string   `xml:"product_name"`
	ProductBrand        string   `xml:"product_brand"`
	ProductArchitecture string   `xml:"product_architecture"`
	Uuid                string   `xml:"uuid"`
	VbiosVersion        string   `xml:"vbios_version"`
	GpuPartNumber       string   `xml:"gpu_part_number"`
	GraphicsClock       string   `xml:"graphics_clock"`
	Clocks              GpuClock `xml:"clocks"`
	MaxClocks           GpuClock `xml:"max_clocks"`
}

type GpuClock struct {
	Graphics string `xml:"graphics_clock"`
	Sm       string `xml:"sm_clock"`
	Mem      string `xml:"mem_clock"`
	Video    string `xml:"video_clock"`
}

func GetNvidiaGPUInfo() (*GPUInfo, error) {
	out, err := exec.Command("nvidia-smi", "-q", "-x").Output()
	if err != nil {
		return nil, err
	}
	var info GPUInfo
	if err := xml.Unmarshal(out, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
