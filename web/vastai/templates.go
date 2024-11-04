package vastai

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) GetTemplatesForUser(ctx context.Context, userID int) ([]Template, error) {
	resp, err := doJSON[struct {
		Success   bool       `json:"success"`
		Templates []Template `json:"template"`
	}](
		ctx, c, http.MethodGet,
		fmt.Sprintf("/v0/users/%d/created-templates/", userID),
		http.StatusOK,
	)
	if err != nil {
		return nil, err
	}

	return resp.Templates, nil
}

type Template struct {
	ID                   int     `json:"id"`
	CreatorID            int     `json:"creator_id"`
	ArgsStr              string  `json:"args_str"`
	Command              any     `json:"command"`
	DefaultTag           string  `json:"default_tag"`
	Desc                 string  `json:"desc"`
	DockerLoginRepo      any     `json:"docker_login_repo"`
	Env                  string  `json:"env"`
	ExtraFilters         string  `json:"extra_filters"`
	Href                 string  `json:"href"`
	Image                string  `json:"image"`
	JupyterDir           any     `json:"jupyter_dir"`
	JupDirect            bool    `json:"jup_direct"`
	JupyterTested        bool    `json:"jupyter_tested"`
	JupyterlabTested     bool    `json:"jupyterlab_tested"`
	MaxCuda              any     `json:"max_cuda"`
	MinCuda              any     `json:"min_cuda"`
	Onstart              string  `json:"onstart"`
	PythonUtf8           bool    `json:"python_utf8"`
	LangUtf8             bool    `json:"lang_utf8"`
	Repo                 string  `json:"repo"`
	Runtype              string  `json:"runtype"`
	SSHDirect            bool    `json:"ssh_direct"`
	Tag                  string  `json:"tag"`
	UseJupyterLab        bool    `json:"use_jupyter_lab"`
	UseSSH               bool    `json:"use_ssh"`
	HashID               string  `json:"hash_id"`
	Name                 string  `json:"name"`
	CreatedFrom          any     `json:"created_from"`
	CreatedFromID        int     `json:"created_from_id"`
	CountCreated         int     `json:"count_created"`
	DescCount            int     `json:"desc_count"`
	RecentCreateDate     float64 `json:"recent_create_date"`
	CreatedAt            float64 `json:"created_at"`
	Private              bool    `json:"private"`
	ReadmeVisible        bool    `json:"readme_visible"`
	ReadmeHash           any     `json:"readme_hash"`
	RecommendedDiskSpace float64 `json:"recommended_disk_space"`
	Recommended          bool    `json:"recommended"`
	Cached               bool    `json:"cached"`
	Autoscaler           bool    `json:"autoscaler"`
}
