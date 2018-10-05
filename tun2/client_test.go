package tun2

import (
	"net"
	"testing"
)

func TestNewClientNullConfig(t *testing.T) {
	_, err := NewClient(nil)
	if err == nil {
		t.Fatalf("expected NewClient(nil) to fail, got non-failure")
	}
}

func TestSmuxListenerIsNetListener(t *testing.T) {
	var sl interface{} = &smuxListener{}
	_, ok := sl.(net.Listener)
	if !ok {
		t.Fatalf("smuxListener does not implement net.Listener")
	}
}
