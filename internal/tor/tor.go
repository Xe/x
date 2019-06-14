// Package tor manages and automates starting a child tor process for exposing TCP services into onionland.
package tor

import (
	"crypto/rsa"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/Yawning/bulb"
	"github.com/phayes/freeport"
	"github.com/sycamoreone/orc/tor"
)

// Config is a wrapper struct for tor configuration.
type Config struct {
	DataDir               string
	HashedControlPassword string
	ClearPassword         string
	Timeout               time.Duration
}

// StartTor starts a new instance of tor or doesn't with the reason why.
func StartTor(cfg Config) (*Tor, error) {
	tc := tor.NewConfig()
	tc.Set("DataDirectory", cfg.DataDir)
	tc.Set("HashedControlPassword", cfg.HashedControlPassword)
	tc.Set("SocksPort", "0")

	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}

	tc.Set("ControlPort", fmt.Sprint(port))

	tc.Timeout = cfg.Timeout

	tcmd, err := tor.NewCmd(tc)
	if err != nil {
		return nil, err
	}

	err = tcmd.Start()
	if err != nil {
		return nil, err
	}

	log.Println("tor started, sleeping for a few seconds for it to settle...")
	time.Sleep(5 * time.Second)

	bc, err := bulb.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}

	err = bc.Authenticate(cfg.ClearPassword)
	if err != nil {
		return nil, err
	}

	t := &Tor{
		tc:   tc,
		tcmd: tcmd,
		bc:   bc,
	}

	return t, nil
}

// Tor is a higher level wrapper to a child tor process
type Tor struct {
	tc   *tor.Config
	tcmd *tor.Cmd
	bc   *bulb.Conn
}

// AddOnion adds an onion service to this machine with the given private key
// (can be nil for an auto-generated key), virtual onion port and TCP destunation.
func (t *Tor) AddOnion(pKey *rsa.PrivateKey, virtPort uint16, destination string) (*bulb.OnionInfo, error) {
	return t.bc.AddOnion([]bulb.OnionPortSpec{
		{
			VirtPort: virtPort,
			Target:   destination,
		},
	}, pKey, true)
}
