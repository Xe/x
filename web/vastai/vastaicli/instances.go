package vastaicli

import (
	"context"
	"fmt"
	"net"
	"strings"

	"al.essio.dev/pkg/shellescape"
)

type InstanceConfig struct {
	DiskSizeGB  int               `json:"diskSizeGB"`
	DockerImage string            `json:"dockerImage"`
	Environment map[string]string `json:"env"`
	OnStartCMD  string            `json:"onStartCmd"`
	Ports       []int             `json:"ports"`
}

func (ic InstanceConfig) EnvString() string {
	var sb strings.Builder

	for _, port := range ic.Ports {
		fmt.Fprintf(&sb, "-p %d:%d ", port, port)
	}

	// -e FOO=bar
	for k, v := range ic.Environment {
		fmt.Fprintf(&sb, "-e %s=%s ", shellescape.Quote(k), shellescape.Quote(v))
	}

	return sb.String()
}

type NewInstance struct {
	Success     bool `json:"success"`
	NewContract int  `json:"new_contract"`
}

func Mint(ctx context.Context, askContractID int, ic InstanceConfig) (*NewInstance, error) {
	result, err := runJSON[NewInstance](
		ctx,
		"vastai", "create", "instance",
		askContractID,
		"--image", ic.DockerImage,
		"--env", ic.EnvString(),
		"--disk", ic.DiskSizeGB,
		"--onstart-cmd", ic.OnStartCMD,
		"--raw",
	)

	if err != nil {
		return nil, fmt.Errorf("can't create instance for contract ID %d: %v", askContractID, err)
	}

	return &result, nil
}

func Slay(ctx context.Context, instanceID int) error {
	type slayResp struct {
		Success bool `json:"success"`
	}

	if _, err := runJSON[slayResp](ctx, "vastai", "destroy", "instance", instanceID, "--raw"); err != nil {
		return fmt.Errorf("can't slay instance %d: %w", instanceID, err)
	}

	return nil
}

func GetInstance(ctx context.Context, instanceID int) (*Instance, error) {
	result, err := runJSON[Instance](ctx, "vastai", "show", "instance", instanceID, "--raw")
	if err != nil {
		return nil, fmt.Errorf("can't get instance %d: %w", instanceID, err)
	}

	return &result, nil
}

type Instance struct {
	ActualStatus          string                 `json:"actual_status"`
	BundleID              int                    `json:"bundle_id"`
	BwNvlink              float64                `json:"bw_nvlink"`
	ClientRunTime         float64                `json:"client_run_time"`
	ComputeCap            int                    `json:"compute_cap"`
	CPUArch               string                 `json:"cpu_arch"`
	CPUCores              int                    `json:"cpu_cores"`
	CPUCoresEffective     float64                `json:"cpu_cores_effective"`
	CPUName               string                 `json:"cpu_name"`
	CPURAM                int                    `json:"cpu_ram"`
	CPUUtil               float64                `json:"cpu_util"`
	CreditBalance         any                    `json:"credit_balance"`
	CreditDiscount        any                    `json:"credit_discount"`
	CreditDiscountMax     float64                `json:"credit_discount_max"`
	CudaMaxGood           float64                `json:"cuda_max_good"`
	CurState              string                 `json:"cur_state"`
	DirectPortCount       int                    `json:"direct_port_count"`
	DirectPortEnd         int                    `json:"direct_port_end"`
	DirectPortStart       int                    `json:"direct_port_start"`
	DiskBw                float64                `json:"disk_bw"`
	DiskName              string                 `json:"disk_name"`
	DiskSpace             float64                `json:"disk_space"`
	DiskUsage             float64                `json:"disk_usage"`
	DiskUtil              float64                `json:"disk_util"`
	Dlperf                float64                `json:"dlperf"`
	DlperfPerDphtotal     float64                `json:"dlperf_per_dphtotal"`
	DphBase               float64                `json:"dph_base"`
	DphTotal              float64                `json:"dph_total"`
	DriverVersion         string                 `json:"driver_version"`
	Duration              float64                `json:"duration"`
	EndDate               float64                `json:"end_date"`
	External              bool                   `json:"external"`
	ExtraEnv              any                    `json:"extra_env"`
	FlopsPerDphtotal      float64                `json:"flops_per_dphtotal"`
	Geolocation           string                 `json:"geolocation"`
	GpuDisplayActive      bool                   `json:"gpu_display_active"`
	GpuFrac               float64                `json:"gpu_frac"`
	GpuLanes              int                    `json:"gpu_lanes"`
	GpuMemBw              float64                `json:"gpu_mem_bw"`
	GpuName               string                 `json:"gpu_name"`
	GpuRAM                int                    `json:"gpu_ram"`
	GpuTemp               any                    `json:"gpu_temp"`
	GpuTotalram           int                    `json:"gpu_totalram"`
	GpuUtil               any                    `json:"gpu_util"`
	HasAvx                int                    `json:"has_avx"`
	HostID                int                    `json:"host_id"`
	HostRunTime           float64                `json:"host_run_time"`
	HostingType           any                    `json:"hosting_type"`
	ID                    int                    `json:"id"`
	ImageArgs             []any                  `json:"image_args"`
	ImageRuntype          string                 `json:"image_runtype"`
	ImageUUID             string                 `json:"image_uuid"`
	InetDown              float64                `json:"inet_down"`
	InetDownBilled        any                    `json:"inet_down_billed"`
	InetDownCost          float64                `json:"inet_down_cost"`
	InetUp                float64                `json:"inet_up"`
	InetUpBilled          any                    `json:"inet_up_billed"`
	InetUpCost            float64                `json:"inet_up_cost"`
	Instance              InstancePricingDetails `json:"instance"`
	IntendedStatus        string                 `json:"intended_status"`
	InternetDownCostPerTb float64                `json:"internet_down_cost_per_tb"`
	InternetUpCostPerTb   float64                `json:"internet_up_cost_per_tb"`
	IsBid                 bool                   `json:"is_bid"`
	JupyterToken          string                 `json:"jupyter_token"`
	Label                 any                    `json:"label"`
	LocalIpaddrs          string                 `json:"local_ipaddrs"`
	Logo                  string                 `json:"logo"`
	MachineDirSSHPort     int                    `json:"machine_dir_ssh_port"`
	MachineID             int                    `json:"machine_id"`
	MemLimit              any                    `json:"mem_limit"`
	MemUsage              any                    `json:"mem_usage"`
	MinBid                float64                `json:"min_bid"`
	MoboName              string                 `json:"mobo_name"`
	NextState             string                 `json:"next_state"`
	NumGpus               int                    `json:"num_gpus"`
	Onstart               any                    `json:"onstart"`
	OsVersion             string                 `json:"os_version"`
	PciGen                float64                `json:"pci_gen"`
	PcieBw                float64                `json:"pcie_bw"`
	PublicIpaddr          string                 `json:"public_ipaddr"`
	Reliability2          float64                `json:"reliability2"`
	Rentable              bool                   `json:"rentable"`
	Score                 float64                `json:"score"`
	Search                SearchPricing          `json:"search"`
	SSHHost               string                 `json:"ssh_host"`
	SSHIdx                string                 `json:"ssh_idx"`
	SSHPort               int                    `json:"ssh_port"`
	StartDate             float64                `json:"start_date"`
	StaticIP              bool                   `json:"static_ip"`
	StatusMsg             string                 `json:"status_msg"`
	StorageCost           float64                `json:"storage_cost"`
	StorageTotalCost      float64                `json:"storage_total_cost"`
	TemplateHashID        any                    `json:"template_hash_id"`
	TimeRemaining         string                 `json:"time_remaining"`
	TimeRemainingIsbid    string                 `json:"time_remaining_isbid"`
	TotalFlops            float64                `json:"total_flops"`
	UptimeMins            any                    `json:"uptime_mins"`
	Verification          string                 `json:"verification"`
	VmemUsage             any                    `json:"vmem_usage"`
	VramCostperhour       float64                `json:"vram_costperhour"`
	Webpage               any                    `json:"webpage"`

	Ports map[string][]PortData
}

func (i Instance) AddrFor(port int) (string, bool) {
	data, ok := i.Ports[fmt.Sprintf("%d/tcp", port)]
	if !ok {
		return "", false
	}

	return net.JoinHostPort(i.PublicIpaddr, data[0].HostPort), true
}

type PortData struct {
	HostIp   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}
