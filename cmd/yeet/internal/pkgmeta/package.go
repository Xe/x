package pkgmeta

type Package struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Homepage    string   `json:"homepage"`
	Group       string   `json:"group"`
	License     string   `json:"license"`
	Goarch      string   `json:"goarch"`
	Replaces    []string `json:"replaces"`
	Depends     []string `json:"depends"`
	Recommends  []string `json:"recommends"`

	EmptyDirs     []string          `json:"emptyDirs"`     // rpm destination path
	ConfigFiles   map[string]string `json:"configFiles"`   // repo-relative source path, rpm destination path
	Documentation map[string]string `json:"documentation"` // repo-relative source path, file in /usr/share/doc/$Name

	Build func(out string) `json:"build"`
}
