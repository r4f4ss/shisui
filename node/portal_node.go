package node

import (
	"github.com/ethereum/go-ethereum/portalnetwork/ethapi"
	"github.com/ethereum/go-ethereum/portalnetwork/web3"
	"github.com/ethereum/go-ethereum/rpc"
)

// Creates a new rpc node for Portal Network client.
func PortalNewHTTP(config PortalApiConfig) error {

	rpcServer := newHTTPServer(config.Log, config.HTTPTimeouts)

	if err := rpcServer.setListenAddr(config.HTTPHost, config.HTTPPort); err != nil {
		return err
	}

	rpcConfig := rpcEndpointConfig{
		batchItemLimit:         config.BatchRequestLimit,
		batchResponseSizeLimit: config.BatchResponseMaxSize,
	}

	var apis = []rpc.API{{
		Namespace: "eth",
		Service:   ethapi.NewPortalEthereumAPI(),
	}, {
		Namespace: "web3",
		Service:   &web3.API{},
	}}
	if err := rpcServer.enableRPC(apis, httpConfig{
		CorsAllowedOrigins: config.HTTPCors,
		Vhosts:             config.HTTPVirtualHosts,
		Modules:            []string{"eth", "web3"},
		prefix:             config.HTTPPathPrefix,
		rpcEndpointConfig:  rpcConfig,
	}); err != nil {
		return err
	}

	if err := rpcServer.start(); err != nil {
		return err
	}

	return nil
}
