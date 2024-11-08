package vastaicli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// zilch returns the zero value of a given type.
func zilch[T any]() T { return *new(T) }

func runJSON[T any](ctx context.Context, program string, args ...any) (T, error) {
	exePath, err := exec.LookPath(program)
	if err != nil {
		return zilch[T](), fmt.Errorf("can't find %s: %w", program, err)
	}

	var argStr []string

	for _, arg := range args {
		argStr = append(argStr, fmt.Sprint(arg))
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, exePath, argStr...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		os.Stderr.Write(stderr.Bytes())
		return zilch[T](), fmt.Errorf("can't run %s: %w", program, err)
	}

	var result T
	if err := json.NewDecoder(&stdout).Decode(&result); err != nil {
		return zilch[T](), fmt.Errorf("can't decode json: %w", err)
	}

	return result, nil
}

func Search(ctx context.Context, filters, orderBy string) ([]SearchItem, error) {
	result, err := runJSON[[]SearchItem](ctx, "vastai", "search", "offers", filters, "-o", orderBy, "--raw")
	if err != nil {
		return nil, fmt.Errorf("while searching for %s: %w", filters, err)
	}

	return result, nil
}

type SearchItem struct {
	AskContractID         int                    `json:"ask_contract_id"`
	BundleID              int                    `json:"bundle_id"`
	BundledResults        any                    `json:"bundled_results"`
	BwNvlink              float64                `json:"bw_nvlink"`
	ComputeCap            int                    `json:"compute_cap"`
	CPUArch               string                 `json:"cpu_arch"`
	CPUCores              int                    `json:"cpu_cores"`
	CPUCoresEffective     float64                `json:"cpu_cores_effective"`
	CPUGhz                float64                `json:"cpu_ghz"`
	CPUName               string                 `json:"cpu_name"`
	CPURAM                int                    `json:"cpu_ram"`
	CreditDiscountMax     float64                `json:"credit_discount_max"`
	CudaMaxGood           float64                `json:"cuda_max_good"`
	DirectPortCount       int                    `json:"direct_port_count"`
	DiscountRate          any                    `json:"discount_rate"`
	DiscountedDphTotal    float64                `json:"discounted_dph_total"`
	DiscountedHourly      float64                `json:"discounted_hourly"`
	DiskBw                float64                `json:"disk_bw"`
	DiskName              string                 `json:"disk_name"`
	DiskSpace             float64                `json:"disk_space"`
	Dlperf                float64                `json:"dlperf"`
	DlperfPerDphtotal     float64                `json:"dlperf_per_dphtotal"`
	DphBase               float64                `json:"dph_base"`
	DphTotal              float64                `json:"dph_total"`
	DphTotalAdj           float64                `json:"dph_total_adj"`
	DriverVers            int                    `json:"driver_vers"`
	DriverVersion         string                 `json:"driver_version"`
	Duration              float64                `json:"duration"`
	EndDate               float64                `json:"end_date"`
	External              any                    `json:"external"`
	FlopsPerDphtotal      float64                `json:"flops_per_dphtotal"`
	Geolocation           string                 `json:"geolocation"`
	Geolocode             float64                `json:"geolocode"`
	GpuArch               string                 `json:"gpu_arch"`
	GpuDisplayActive      bool                   `json:"gpu_display_active"`
	GpuFrac               float64                `json:"gpu_frac"`
	GpuIds                []int                  `json:"gpu_ids"`
	GpuLanes              int                    `json:"gpu_lanes"`
	GpuMaxPower           float64                `json:"gpu_max_power"`
	GpuMaxTemp            float64                `json:"gpu_max_temp"`
	GpuMemBw              float64                `json:"gpu_mem_bw"`
	GpuName               string                 `json:"gpu_name"`
	GpuRAM                float64                `json:"gpu_ram"`
	GpuTotalRAM           float64                `json:"gpu_total_ram"`
	HasAvx                int                    `json:"has_avx"`
	HostID                int                    `json:"host_id"`
	HostingType           int                    `json:"hosting_type"`
	Hostname              any                    `json:"hostname"`
	ID                    int                    `json:"id"`
	InetDown              float64                `json:"inet_down"`
	InetDownCost          float64                `json:"inet_down_cost"`
	InetUp                float64                `json:"inet_up"`
	InetUpCost            float64                `json:"inet_up_cost"`
	Instance              InstancePricingDetails `json:"instance"`
	InternetDownCostPerTb float64                `json:"internet_down_cost_per_tb"`
	InternetUpCostPerTb   float64                `json:"internet_up_cost_per_tb"`
	IsBid                 bool                   `json:"is_bid"`
	Logo                  string                 `json:"logo"`
	MachineID             int                    `json:"machine_id"`
	MinBid                float64                `json:"min_bid"`
	MoboName              string                 `json:"mobo_name"`
	NumGpus               int                    `json:"num_gpus"`
	OsVersion             string                 `json:"os_version"`
	PciGen                float64                `json:"pci_gen"`
	PcieBw                float64                `json:"pcie_bw"`
	PublicIpaddr          string                 `json:"public_ipaddr"`
	Reliability           float64                `json:"reliability"`
	Reliability2          float64                `json:"reliability2"`
	ReliabilityMult       float64                `json:"reliability_mult"`
	Rentable              bool                   `json:"rentable"`
	Rented                bool                   `json:"rented"`
	Rn                    int                    `json:"rn"`
	Score                 float64                `json:"score"`
	Search                SearchPricing          `json:"search"`
	StartDate             float64                `json:"start_date"`
	StaticIP              bool                   `json:"static_ip"`
	StorageCost           float64                `json:"storage_cost"`
	StorageTotalCost      float64                `json:"storage_total_cost"`
	TimeRemaining         string                 `json:"time_remaining"`
	TimeRemainingIsbid    string                 `json:"time_remaining_isbid"`
	TotalFlops            float64                `json:"total_flops"`
	Vericode              int                    `json:"vericode"`
	Verification          string                 `json:"verification"`
	VmsEnabled            bool                   `json:"vms_enabled"`
	VramCostperhour       float64                `json:"vram_costperhour"`
	Webpage               any                    `json:"webpage"`
}

type InstancePricingDetails struct {
	DiscountTotalHour      float64 `json:"discountTotalHour"`
	DiscountedTotalPerHour float64 `json:"discountedTotalPerHour"`
	DiskHour               float64 `json:"diskHour"`
	GpuCostPerHour         float64 `json:"gpuCostPerHour"`
	TotalHour              float64 `json:"totalHour"`
}

type SearchPricing struct {
	DiscountTotalHour      float64 `json:"discountTotalHour"`
	DiscountedTotalPerHour float64 `json:"discountedTotalPerHour"`
	DiskHour               float64 `json:"diskHour"`
	GpuCostPerHour         float64 `json:"gpuCostPerHour"`
	TotalHour              float64 `json:"totalHour"`
}

type Foo struct {
	SomeValue    string `json:"someValue"`
	AnotherValue string "json:\"anotherValue\""
}
