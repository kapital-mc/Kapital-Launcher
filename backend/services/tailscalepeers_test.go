package services

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"testing"

	"kapital/backend/models"
)

const statusWithPeers = `{"BackendState":"Running","Version":"1.104.1",
 "Self":{"TailscaleIPs":["100.70.0.1"]},
 "Peer":{"nodekey:a":{"HostName":"server","TailscaleIPs":["100.64.209.106","fd7a:115c:a1e0::1"]},
         "nodekey:b":{"TailscaleIPs":["not an address"]}}}`

func TestDetectHandsOverThePeersOnlyWhileRunning(t *testing.T) {
	cli := fakeCLI("linux", nil, "/usr/bin/tailscale", func(...string) ([]byte, error) {
		return []byte(statusWithPeers), nil
	})
	info, peers := cli.Detect(context.Background())
	if info.State != TailscaleRunning || len(peers) != 2 {
		t.Fatalf("%+v %v", info, peers)
	}
	if _, ok := peers[netip.MustParseAddr("100.70.0.1")]; ok {
		t.Error("this machine's own address is a peer")
	}
	stopped := fakeCLI("linux", nil, "/usr/bin/tailscale", func(...string) ([]byte, error) {
		return []byte(`{"BackendState":"Stopped","Peer":{"k":{"TailscaleIPs":["100.64.209.106"]}}}`), nil
	})
	if _, peers := stopped.Detect(context.Background()); peers != nil {
		t.Errorf("a stopped Tailscale has peers: %v", peers)
	}
}

func TestTailscaleConnectedComparesTheResolvedAddress(t *testing.T) {
	c := withTailscale()
	c.Server.Addresses[1].Address = "ts.example:25566"
	chapters := []models.Chapter{c, {ID: "luxemburg"}}
	peers := TailscalePeers{netip.MustParseAddr("100.64.209.106"): {}}
	var asked []string
	lookup := func(_ context.Context, host string) ([]netip.Addr, error) {
		asked = append(asked, host)
		return []netip.Addr{netip.MustParseAddr("::ffff:100.64.209.106")}, nil
	}
	if got := TailscaleConnected(context.Background(), chapters, peers, lookup); !slices.Equal(got, []string{"frangfurd"}) {
		t.Errorf("connected: %v", got)
	}
	if !slices.Equal(asked, []string{"ts.example"}) {
		t.Errorf("looked up %v, want the host without its port", asked)
	}
	other := TailscalePeers{netip.MustParseAddr("100.64.0.9"): {}}
	if got := TailscaleConnected(context.Background(), chapters, other, lookup); len(got) != 0 {
		t.Errorf("another peer: %v", got)
	}
	failing := func(context.Context, string) ([]netip.Addr, error) { return nil, errors.New("no such host") }
	if got := TailscaleConnected(context.Background(), chapters, peers, failing); got == nil || len(got) != 0 {
		t.Errorf("unresolved: %v", got)
	}
	if got := TailscaleConnected(context.Background(), chapters, nil, lookup); got == nil || len(got) != 0 {
		t.Errorf("no peers: %v", got)
	}
}
