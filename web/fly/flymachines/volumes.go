package flymachines

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Volume struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	State             string    `json:"state"`
	SizeGB            int       `json:"size_gb"`
	Region            string    `json:"region"`
	Zone              string    `json:"zone"`
	Encrypted         bool      `json:"encrypted"`
	AttachedAllocID   string    `json:"attached_alloc_id"`
	AttachedMachineID string    `json:"attached_machine_id"`
	CreatedAt         time.Time `json:"created_at"`
	Blocks            int       `json:"blocks"`
	BlockSize         int       `json:"block_size"`
	BlocksAvail       int       `json:"blocks_avail"`
	BlocksFree        int       `json:"blocks_free"`
	FSType            string    `json:"fs_type"`
	SnapshotRetention int       `json:"snapshot_retention"`
	HostDedicationKey string    `json:"host_dedication_key,omitempty"`
}

func (c *Client) GetVolumes(ctx context.Context, appName string) ([]Volume, error) {
	return doJSON[[]Volume](ctx, c, http.MethodGet, "/v1/apps/"+appName+"/volumes", http.StatusOK)
}

func (c *Client) GetVolume(ctx context.Context, appName, volumeID string) (*Volume, error) {
	result, err := doJSON[Volume](ctx, c, http.MethodGet, "/v1/apps/"+appName+"/volumes/"+volumeID, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't decode GetVolume response: %w", err)
	}

	return &result, nil
}

type CreateVolume struct {
	Compute           *MachineGuest `json:"compute,omitempty"`
	Encrypted         bool          `json:"encrypted,omitempty"`
	FSType            string        `json:"fs_type,omitempty"`
	MachinesOnly      bool          `json:"machines_only,omitempty"`
	Name              string        `json:"name,omitempty"`
	Region            string        `json:"region,omitempty"`
	RequireUniqueZone bool          `json:"require_unique_zone"`
	SizeGB            int           `json:"size_gb,omitempty"`
	SnapshotID        string        `json:"snapshot_id,omitempty"`
	SnapshotRetention int           `json:"snapshot_retention"`
	SourceVolumeID    string        `json:"source_volume_id,omitempty"`
}

func (cv CreateVolume) Fork(vol *Volume) CreateVolume {
	cv.SourceVolumeID = vol.ID
	cv.SnapshotRetention = vol.SnapshotRetention
	cv.FSType = vol.FSType
	return cv
}

func (c *Client) CreateVolume(ctx context.Context, appName string, cv CreateVolume) (*Volume, error) {
	result, err := doJSONBody[CreateVolume, Volume](ctx, c, http.MethodPost, "/v1/apps/"+appName+"/volumes", cv, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't decode CreateVolume response: %w", err)
	}

	return &result, nil
}

func (c *Client) DeleteVolume(ctx context.Context, appName, volumeID string) error {
	err := c.doRequestNoResponse(ctx, http.MethodDelete, "/v1/apps/"+appName+"/volumes/"+volumeID)
	if err != nil {
		return err
	}

	return nil
}

type ExtendVolumeResponse struct {
	NeedsRestart bool   `json:"needs_restart"`
	Volume       Volume `json:"volume"`
}

func (c *Client) ExtendVolume(ctx context.Context, appName, voluleID string, sizeGB int) (*ExtendVolumeResponse, error) {
	type req struct {
		SizeGB int `json:"size_gb"`
	}

	result, err := doJSONBody[req, ExtendVolumeResponse](ctx, c, http.MethodPost, "/v1/apps/"+appName+"/volumes/"+voluleID+"/extend", req{sizeGB}, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("flymachines: can't decode ExtendVolume response: %w", err)
	}

	return &result, nil
}

type Snapshot struct {
	CreatedAt time.Time `json:"created_at"`
	Digest    string    `json:"digest"`
	ID        string    `json:"id"`
	Size      int       `json:"size"`
}

func (c *Client) ListVolumeSnapshots(ctx context.Context, appName, volumeID string) ([]Snapshot, error) {
	return doJSON[[]Snapshot](ctx, c, http.MethodGet, "/v1/apps/"+appName+"/volumes/"+volumeID+"/snapshots", http.StatusOK)
}
