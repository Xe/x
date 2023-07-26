package main

type IRC struct {
	Server   string `json:"server"`
	Password string `json:"password"`
	Channel  string `json:"channel"`
	Regex    string `json:"regex"`
	Nick     string `json:"nick"`
	User     string `json:"user"`
	Real     string `json:"real"`
}

type Show struct {
	Title    string `json:"title"`
	DiskPath string `json:"diskPath"`
	Quality  string `json:"quality"`
}

type Transmission struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
	HTTPS    bool   `json:"https"`
	RPCURI   string `json:"rpcURI"`
}

type Config struct {
	IRC          IRC          `json:"irc"`
	Transmission Transmission `json:"transmission"`
	Shows        []Show       `json:"shows"`
	RSSKey       string       `json:"rssKey"`
}
