package node

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type PortalApiConfig struct {
	Log                  log.Logger
	ApiHttpEnabled       bool
	HTTPHost             string
	HTTPPort             int
	HTTPCors             []string
	HTTPVirtualHosts     []string
	HTTPModules          []string
	HTTPTimeouts         rpc.HTTPTimeouts
	HTTPPathPrefix       string
	BatchRequestLimit    int
	BatchResponseMaxSize int
}
