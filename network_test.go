package eltest_test

import (
	"testing"

	"github.com/ttab/eltest"
)

func TestGetNetworkAndGateway(t *testing.T) {
	network, err := eltest.GetNetwork()
	eltest.Must(t, err, "get network")

	if network.Name == "" || network.ID == "" {
		t.Fatal("network must have a name and ID")
	}

	gw, err := eltest.GetGatewayIP()
	eltest.Must(t, err, "get gateway IP")

	if len(gw) == 0 {
		t.Fatal("a valid IP must be returned")
	}
}
