package bcasthttp

import (
	"testing"
	"time"

	"github.com/advanderveer/go-test"
)

var _ AddrDetector = &ipifyDetector{}

func TestIpifyDetector(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	d, ok := DynamicAddr().(*ipifyDetector)
	test.Equals(t, ok, true) //should be the ipify detector

	ip1 := d.DetectExternalIP(true)
	if ip1 == nil {
		t.Skip("failed to detect external ip, assuming servers are offline")
	}

	t0 := time.Now()
	ip2 := d.DetectExternalIP(true)
	test.Equals(t, ip1, ip2)
	test.Assert(t, time.Now().Sub(t0) < time.Millisecond, "should be read from cache")
}
