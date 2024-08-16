package ethapi

import (
	"runtime"
)

type PortalEthereumAPI struct {
}

// NewPotalEthereumAPI creates a new Ethereum protocol API.
func NewPortalEthereumAPI() *PortalEthereumAPI {
	return &PortalEthereumAPI{}
}

func (p *PortalEthereumAPI) ClientVersion() string {
	// TODO add version
	name := "ethApi-Shisui"
	name += "/" + runtime.GOOS + "-" + runtime.GOARCH
	name += "/" + runtime.Version()
	return name
}
