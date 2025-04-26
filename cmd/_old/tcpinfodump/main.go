package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/mikioh/tcp"
	"github.com/mikioh/tcpinfo"
)

func main() {
	c, err := net.Dial("tcp", "golang.org:80")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	tc, err := tcp.NewConn(c)
	if err != nil {
		log.Fatal(err)
	}
	var o tcpinfo.Info
	var b [256]byte
	i, err := tc.Option(o.Level(), o.Name(), b[:])
	if err != nil {
		log.Fatal(err)
	}
	txt, err := json.Marshal(i)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(txt))
}
