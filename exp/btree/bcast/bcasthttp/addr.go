package bcasthttp

import (
	"io/ioutil"
	"net"
	"net/http"
)

// AddrDetector detects ip address. The implementations should serve it from
// a cache if possible.
type AddrDetector interface {
	DetectExternalIP(cached bool) net.IP
}

// StaticAddr returns an address detector that returns a fixed ip
func StaticAddr(ip net.IP) AddrDetector { return fixedDetector(ip) }

type fixedDetector net.IP

func (d fixedDetector) DetectExternalIP(cached bool) (ip net.IP) { return net.IP(d) }

// DynamicAddr returns an address detector that works on dynamic external addresses
func DynamicAddr() AddrDetector { return &ipifyDetector{} }

type ipifyDetector struct{ ip net.IP }

func (d *ipifyDetector) DetectExternalIP(cached bool) (ip net.IP) {
	if cached && d.ip != nil {
		return d.ip
	}

	resp, err := http.Get("https://api6.ipify.org")
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil //something went wrong
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil //reading went wrong
	}

	d.ip = net.ParseIP(string(data))
	return d.ip
}
