package vastai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) SearchByTemplate(ctx context.Context, t Template) ([]Offer, error) {
	filters := map[string]any{}
	if err := json.Unmarshal([]byte(t.ExtraFilters), &filters); err != nil {
		return nil, fmt.Errorf("can't unmarshal extra filters for template %s (%d): %w", t.Name, t.ID, err)
	}
	filters["order"] = []string{"dphtotal", "asc"}

	extraFilters, err := json.Marshal(filters)
	if err != nil {
		return nil, fmt.Errorf("can't remarshal extra filters for template %s (%d): %w", t.Name, t.ID, err)
	}

	q := url.Values{}
	q.Set("q", string(extraFilters))

	resp, err := doJSON[struct {
		Offers []Offer `json:"offers"`
	}](
		ctx, c, http.MethodGet,
		fmt.Sprintf("/v0/bundles/%s", q.Encode()),
		http.StatusOK,
	)
	if err != nil {
		return nil, fmt.Errorf("can't find instances with template %s (%d): %w", t.Name, t.ID, err)
	}

	return resp.Offers, nil
}

type Offer struct {
	IsBid             bool    `json:"is_bid"`
	InetUpBilled      any     `json:"inet_up_billed"`
	InetDownBilled    any     `json:"inet_down_billed"`
	External          bool    `json:"external"`
	Webpage           any     `json:"webpage"`
	Logo              string  `json:"logo"`
	Rentable          bool    `json:"rentable"`
	ComputeCap        int     `json:"compute_cap"`
	DriverVersion     string  `json:"driver_version"`
	CudaMaxGood       int     `json:"cuda_max_good"`
	MachineID         int     `json:"machine_id"`
	HostingType       any     `json:"hosting_type"`
	PublicIpaddr      string  `json:"public_ipaddr"`
	Geolocation       string  `json:"geolocation"`
	FlopsPerDphtotal  float64 `json:"flops_per_dphtotal"`
	DlperfPerDphtotal float64 `json:"dlperf_per_dphtotal"`
	Reliability2      float64 `json:"reliability2"`
	HostRunTime       int     `json:"host_run_time"`
	HostID            int     `json:"host_id"`
	ID                int     `json:"id"`
	BundleID          int     `json:"bundle_id"`
	NumGpus           int     `json:"num_gpus"`
	TotalFlops        float64 `json:"total_flops"`
	MinBid            float64 `json:"min_bid"`
	DphBase           float64 `json:"dph_base"`
	DphTotal          float64 `json:"dph_total"`
	GpuName           string  `json:"gpu_name"`
	GpuRAM            int     `json:"gpu_ram"`
	GpuDisplayActive  bool    `json:"gpu_display_active"`
	GpuMemBw          float64 `json:"gpu_mem_bw"`
	BwNvlink          int     `json:"bw_nvlink"`
	DirectPortCount   int     `json:"direct_port_count"`
	GpuLanes          int     `json:"gpu_lanes"`
	PcieBw            float64 `json:"pcie_bw"`
	PciGen            int     `json:"pci_gen"`
	Dlperf            float64 `json:"dlperf"`
	CPUName           string  `json:"cpu_name"`
	MoboName          string  `json:"mobo_name"`
	CPURAM            int     `json:"cpu_ram"`
	CPUCores          int     `json:"cpu_cores"`
	CPUCoresEffective int     `json:"cpu_cores_effective"`
	GpuFrac           int     `json:"gpu_frac"`
	HasAvx            int     `json:"has_avx"`
	DiskSpace         float64 `json:"disk_space"`
	DiskName          string  `json:"disk_name"`
	DiskBw            float64 `json:"disk_bw"`
	InetUp            float64 `json:"inet_up"`
	InetDown          float64 `json:"inet_down"`
	StartDate         float64 `json:"start_date"`
	EndDate           any     `json:"end_date"`
	Duration          any     `json:"duration"`
	StorageCost       float64 `json:"storage_cost"`
	InetUpCost        float64 `json:"inet_up_cost"`
	InetDownCost      float64 `json:"inet_down_cost"`
	StorageTotalCost  int     `json:"storage_total_cost"`
	Verification      string  `json:"verification"`
	Score             float64 `json:"score"`
	Rented            bool    `json:"rented"`
	BundledResults    int     `json:"bundled_results"`
	PendingCount      int     `json:"pending_count"`
}
