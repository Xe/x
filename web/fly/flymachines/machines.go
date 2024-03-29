package flymachines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func Ptr[T any](t T) *T {
	return &t
}

type MachineRestartPolicy string

const (
	MachineRestartPolicyNo        MachineRestartPolicy = "no"
	MachineRestartPolicyAlways    MachineRestartPolicy = "always"
	MachineRestartPolicyOnFailure MachineRestartPolicy = "on-failure"
)

type MachineConfig struct {
	Env        map[string]string `json:"env"`
	Metadata   map[string]string `json:"metadata"`
	Mounts     []MachineMount    `json:"mounts,omitempty"`
	Image      string            `json:"image"`
	Restart    MachineRestart    `json:"restart"`
	Guest      MachineGuest      `json:"guest"`
	StopConfig MachineStopConfig `json:"stop_config"`
	Processes  []MachineProcess  `json:"processes,omitempty"`
}

type MachineProcess struct {
	Cmd        []string          `json:"cmd,omitempty"`
	Entrypoint []string          `json:"entrypoint,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Exec       []string          `json:"exec,omitempty"`
	User       string            `json:"user,omitempty"`
}

type MachineMount struct {
	Encrypted bool   `json:"encrypted"`
	Path      string `json:"path"`
	SizeGB    int    `json:"size_gb"`
	Volume    string `json:"volume"`
	Name      string `json:"name"`
}

type MachineService struct {
	Protocol                 string                    `json:"protocol"`
	InternalPort             int                       `json:"internal_port"`
	ForceInstanceDescription *string                   `json:"force_instance_description,omitempty"`
	ForceInstanceKey         *string                   `json:"force_instance_key,omitempty"`
	Ports                    []MachinePort             `json:"ports"`
	Checks                   []MachineCheck            `json:"checks"`
	MinMachinesRunning       int                       `json:"min_machines_running"`
	Concurrency              MachineServiceConcurrency `json:"concurrency"`
}

type MachineGuest struct {
	CPUKind          string   `json:"cpu_kind"` // "shared" or "performance"
	CPUs             int      `json:"cpus"`
	MemoryMB         int      `json:"memory_mb"`
	GPUKind          string   `json:"gpu_kind,omitempty"`
	KernelArgs       []string `json:"kernel_args,omitempty"`
	HostDedicationID string   `json:"host_dedication_id,omitempty"`
}

type MachineStopConfig struct {
	Timeout string `json:"timeout"`
	Signal  string `json:"signal"`
}

type MachineRestart struct {
	MaxRetries int                  `json:"max_retries"` // only relevant when Policy is "on-fail"
	Policy     MachineRestartPolicy `json:"policy"`
}

type MachineServiceConcurrency struct {
	Type      string `json:"type"`
	HardLimit int    `json:"hard_limit"`
	SoftLimit int    `json:"soft_limit"`
}

type MachineCheck struct {
	Type          string              `json:"type"`
	Interval      time.Duration       `json:"interval"`
	Timeout       time.Duration       `json:"timeout"`
	GracePeriod   time.Duration       `json:"grace_period"`
	Path          string              `json:"path"`
	TLSServerName *string             `json:"tls_server_name,omitempty"`
	TLSSkipVerify *bool               `json:"tls_skip_verify,omitempty"`
	Headers       []MachineHTTPHeader `json:"headers,omitempty"`
}

type MachineHTTPHeader struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type MachinePort struct {
	Port       int      `json:"port"`
	Handlers   []string `json:"handlers"`
	ForceHTTPS *bool    `json:"force_https,omitempty"`
	StartPort  *int     `json:"start_port,omitempty"`
	EndPort    *int     `json:"end_port,omitempty"`
}

type ImageRef struct {
	Registry   string          `json:"registry"`
	Repository string          `json:"repository"`
	Tag        string          `json:"tag"`
	Digest     string          `json:"digest"`
	Labels     json.RawMessage `json:"labels"` // TODO(Xe): figure out what this is
}

type MachineEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Status    string          `json:"status"`
	Source    string          `json:"source"`
	Timestamp MilliTime       `json:"timestamp"`
	Request   json.RawMessage `json:"request"` // Request can be anything, so we just store it as a raw message
}

type CheckStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Output    string    `json:"output"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Machine struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	State      string         `json:"state"`
	Region     string         `json:"region"`
	InstanceID string         `json:"instance_id"`
	PrivateIP  string         `json:"private_ip"`
	Config     MachineConfig  `json:"config"`
	ImageRef   ImageRef       `json:"image_ref"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  *time.Time     `json:"updated_at"`
	Events     []MachineEvent `json:"events"`
	Checks     []CheckStatus  `json:"checks"`
}

func (c *Client) GetAppMachines(ctx context.Context, appName string) ([]Machine, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/v1/apps/"+appName+"/machines", nil)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't create GetAppMachines request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't do GetAppMachines request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("flymachines: GetAppMachines request failed: %s", resp.Status)
	}

	var machines []Machine
	if err := json.NewDecoder(resp.Body).Decode(&machines); err != nil {
		return nil, fmt.Errorf("flymachines: can't decode GetAppMachines response: %w", err)
	}

	return machines, nil
}

type CreateMachine struct {
	Config                  MachineConfig `json:"config"`
	LeaseTTL                int           `json:"lease_ttl"`
	LSVD                    bool          `json:"lsvd"` // should be true?
	Name                    string        `json:"name"`
	Region                  string        `json:"region"`
	SkipLaunch              *bool         `json:"skip_launch,omitempty"`
	SkipServiceRegistration *bool         `json:"skip_service_registration,omitempty"`
}

func (c *Client) CreateMachine(ctx context.Context, appID string, cm CreateMachine) (*Machine, error) {
	result, err := doJSONBody[CreateMachine, Machine](ctx, c, http.MethodPost, "/v1/apps/"+appID+"/machines", cm, http.StatusOK)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetAppMachine(ctx context.Context, appID, machineID string) (*Machine, error) {
	result, err := doJSON[Machine](ctx, c, http.MethodGet, "/v1/apps/"+appID+"/machines/"+machineID, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't decode GetAppMachine response: %w", err)
	}

	return &result, nil
}

func (c *Client) DeleteAppMachine(ctx context.Context, appID, machineID string) error {
	return c.doRequestNoResponse(ctx, http.MethodDelete, "/v1/apps/"+appID+"/machines/"+machineID)
}

func (c *Client) DestroyAppMachine(ctx context.Context, appID, machineID string) error {
	return c.doRequestNoResponse(ctx, http.MethodDelete, "/v1/apps/"+appID+"/machines/"+machineID+"?force=true")
}

func (c *Client) CordonAppMachine(ctx context.Context, appID, machineID string) error {
	return c.doRequestNoResponse(ctx, http.MethodPost, "/v1/apps/"+appID+"/machines/"+machineID+"/cordon")
}

func (c *Client) UncordonAppMachine(ctx context.Context, appID, machineID string) error {
	return c.doRequestNoResponse(ctx, http.MethodPost, "/v1/apps/"+appID+"/machines/"+machineID+"/uncordon")
}

func (c *Client) StartAppMachine(ctx context.Context, appID, machineID string) error {
	return c.doRequestNoResponse(ctx, http.MethodPost, "/v1/apps/"+appID+"/machines/"+machineID+"/start")
}

func (c *Client) StopAppMachine(ctx context.Context, appID, machineID string) error {
	return c.doRequestNoResponse(ctx, http.MethodPost, "/v1/apps/"+appID+"/machines/"+machineID+"/stop")
}

func (c *Client) RestartAppMachine(ctx context.Context, appID, machineID string) error {
	return c.doRequestNoResponse(ctx, http.MethodPost, "/v1/apps/"+appID+"/machines/"+machineID+"/restart")
}

func (c *Client) GetAppMachineEvents(ctx context.Context, appID, machineID string) ([]MachineEvent, error) {
	result, err := doJSON[[]MachineEvent](ctx, c, http.MethodGet, "/v1/apps/"+appID+"/machines/"+machineID+"/events", http.StatusOK)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) GetAppMachineMetadata(ctx context.Context, appID, machineID string) (map[string]string, error) {
	result, err := doJSON[map[string]string](ctx, c, http.MethodGet, "/v1/apps/"+appID+"/machines/"+machineID+"/metadata", http.StatusOK)
	if err != nil {
		return nil, err
	}

	return result, nil
}
