// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// PortStatus is the result of a single port probe.
type PortStatus int

const (
	PortOpen     PortStatus = iota // SYN+ACK received (TCP) or no ICMP unreachable (UDP)
	PortRefused                    // RST received — host reachable but port closed
	PortFiltered                   // Timeout — firewall silently dropping packets
	PortUnknown                    // Probe not yet run or IP missing
)

func (p PortStatus) String() string {
	switch p {
	case PortOpen:
		return "OPEN"
	case PortRefused:
		return "CLOSED"
	case PortFiltered:
		return "FILTERED"
	default:
		return "UNKNOWN"
	}
}

// ProbeTimeout is how long each dial attempt waits before giving up.
const ProbeTimeout = 2 * time.Second

// NodeProbeResult holds per-node, per-port probe outcomes.
type NodeProbeResult struct {
	NodeID    string
	Hostname  string
	Addr      string // IP address probed
	NodeState string // Docker-reported state ("ready", "down", …)
	TCP2377   PortStatus
	TCP7946   PortStatus
	UDP7946   PortStatus
	// UDP4789 is kernel-level VXLAN; not directly probeable from userspace.
}

// ProbeNodePorts runs TCP and UDP probes against a single node IP.
// It is safe to call concurrently.
func ProbeNodePorts(entry NodeEntry) NodeProbeResult {
	ip := entry.Addr
	if ip == "" {
		return NodeProbeResult{
			NodeID:    entry.ID,
			Hostname:  entry.Hostname,
			NodeState: entry.State,
			Addr:      "",
			TCP2377:   PortUnknown,
			TCP7946:   PortUnknown,
			UDP7946:   PortUnknown,
		}
	}

	return NodeProbeResult{
		NodeID:    entry.ID,
		Hostname:  entry.Hostname,
		NodeState: entry.State,
		Addr:      ip,
		TCP2377:   probeTCP(ip, 2377),
		TCP7946:   probeTCP(ip, 7946),
		UDP7946:   probeUDP(ip, 7946),
	}
}

// ProbeAllNodes fans out probes to every node in the snapshot concurrently
// and returns one result per node, in the same order as entries.
func ProbeAllNodes(entries []NodeEntry) []NodeProbeResult {
	results := make([]NodeProbeResult, len(entries))
	var wg sync.WaitGroup
	for i, entry := range entries {
		wg.Add(1)
		go func(idx int, e NodeEntry) {
			defer wg.Done()
			results[idx] = ProbeNodePorts(e)
		}(i, entry)
	}
	wg.Wait()
	return results
}

// probeTCP attempts a TCP connection to host:port.
func probeTCP(host string, port int) PortStatus {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, ProbeTimeout)
	if err == nil {
		conn.Close()
		return PortOpen
	}
	if isRefused(err) {
		return PortRefused
	}
	return PortFiltered
}

// probeUDP sends a zero-byte datagram to host:port and reads with a short
// deadline. On most kernels, a closed UDP port triggers an ICMP port-
// unreachable, which surfaces as a "connection refused" read error.
// Silence (timeout) means the port is open or the ICMP is being dropped.
func probeUDP(host string, port int) PortStatus {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("udp", addr, ProbeTimeout)
	if err != nil {
		return PortFiltered
	}
	defer conn.Close()

	// Send a probe datagram so the kernel has something to respond to.
	_ = conn.SetDeadline(time.Now().Add(ProbeTimeout))
	_, _ = conn.Write([]byte{})

	// Attempt a read; ICMP unreachable turns into a read error.
	buf := make([]byte, 1)
	_, readErr := conn.Read(buf)
	if readErr != nil {
		if isRefused(readErr) {
			return PortRefused
		}
		// Timeout → open or silently filtered
		return PortOpen
	}
	// We actually read data — port is responding.
	return PortOpen
}

// isRefused returns true if the error represents a connection refused.
func isRefused(err error) bool {
	if err == nil {
		return false
	}
	// net.OpError wraps a syscall error; the string is the most portable check.
	e := err.Error()
	return contains(e, "connection refused") || contains(e, "refused")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
