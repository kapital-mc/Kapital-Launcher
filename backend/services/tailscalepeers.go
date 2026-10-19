package services

import (
	"context"
	"net"
	"net/netip"
	"time"

	"kapital/backend/models"
)

// tailscaleLookupTimeout bounds the DNS lookup of one chapter's Tailscale
// address, so a slow resolver cannot hold up the settings page.
const tailscaleLookupTimeout = 2 * time.Second

// TailscalePeers is the set of addresses this machine's Tailscale reaches: its
// peers', which include a machine shared with the player once they accepted
// the share link. It lives for one look and is never logged.
type TailscalePeers map[netip.Addr]struct{}

func newTailscalePeers(peers map[string]struct{ TailscaleIPs []string }) TailscalePeers {
	set := TailscalePeers{}
	for _, p := range peers {
		for _, ip := range p.TailscaleIPs {
			if a, err := netip.ParseAddr(ip); err == nil {
				set[a.Unmap()] = struct{}{}
			}
		}
	}
	return set
}

// LookupNetIP resolves a host, as net.DefaultResolver does; a test swaps it.
type LookupNetIP func(ctx context.Context, host string) ([]netip.Addr, error)

// DefaultLookupNetIP is the system resolver.
func DefaultLookupNetIP(ctx context.Context, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

// TailscaleConnected is the chapters whose Tailscale address is one of this
// machine's peers: Tailscale is running and the player accepted the share
// link, so the address answers here. The address's host is resolved (the
// manifest names it, ts.<domain>) and compared; a host that does not resolve
// is not connected. A shared machine usually keeps its address in the
// player's tailnet, and when it does not, the name does not reach it either.
func TailscaleConnected(ctx context.Context, chapters []models.Chapter, peers TailscalePeers, lookup LookupNetIP) []string {
	connected := []string{}
	if len(peers) == 0 {
		return connected
	}
	for _, c := range chapters {
		addr, ok := TailscaleAddress(c)
		if !ok {
			continue
		}
		host := addr.Address
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if peers.reaches(ctx, host, lookup) {
			connected = append(connected, c.ID)
		}
	}
	return connected
}

func (p TailscalePeers) reaches(ctx context.Context, host string, lookup LookupNetIP) bool {
	ctx, cancel := context.WithTimeout(ctx, tailscaleLookupTimeout)
	defer cancel()
	ips, err := lookup(ctx, host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if _, ok := p[ip.Unmap()]; ok {
			return true
		}
	}
	return false
}
