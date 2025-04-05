package pkgmeta

type Package struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Homepage    string   `json:"homepage"`
	Group       string   `json:"group"`
	License     string   `json:"license"`
	Platform    string   `json:"platform"` // if not set, default to linux
	Goarch      string   `json:"goarch"`
	Replaces    []string `json:"replaces"`
	Depends     []string `json:"depends"`
	Recommends  []string `json:"recommends"`

	EmptyDirs     []string          `json:"emptyDirs"`     // rpm destination path
	ConfigFiles   map[string]string `json:"configFiles"`   // pwd-relative source path, rpm destination path
	Documentation map[string]string `json:"documentation"` // pwd-relative source path, file in /usr/share/doc/$Name
	Files         map[string]string `json:"files"`         // pwd-relative source path, rpm destination path

	Build    func(BuildInput)     `json:"build"`
	Filename func(Package) string `json:"mkFilename"`
}

type BuildInput struct {
	Output  string `json:"out"`
	Bin     string `json:"bin"`
	Doc     string `json:"doc"`
	Etc     string `json:"etc"`
	Man     string `json:"man"`
	Systemd string `json:"systemd"`
}

func (b BuildInput) String() string {
	return b.Output
}
