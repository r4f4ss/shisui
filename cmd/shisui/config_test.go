package main

import (
	"flag"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestGenConfig(t *testing.T) {
	size := uint64(5 * 1000 * 1000 * 1000)
	flagSet := flag.NewFlagSet("test", 0)
	flagSet.String("rpc.addr", "127.0.0.11", "test")
	flagSet.String("rpc.port", "8888", "test")
	flagSet.String("data.dir", "./test", "test")
	flagSet.Uint64("data.capacity", size, "test")
	// flagSet.String("udp.addr", "172.23.50.11", "test")
	flagSet.Int("udp.port", 9999, "test")
	flagSet.Int("loglevel", 3, "test")
	val := cli.NewStringSlice("history")
	flagSet.Var(val, "networks", "test")

	command := &cli.Command{Name: "mycommand"}

	ctx := cli.NewContext(nil, flagSet, nil)
	ctx.Command = command

	config, err := getPortalConfig(ctx)
	require.NoError(t, err)

	require.Equal(t, config.DataCapacity, size)
	require.Equal(t, config.DataDir, "./test")
	require.Equal(t, config.LogLevel, 3)
	// require.Equal(t, config.RpcAddr, "127.0.0.11:8888")
	require.Equal(t, config.Protocol.ListenAddr, ":9999")
	require.Equal(t, config.Networks, []string{"history"})
}

func TestApiConfig(t *testing.T) {

	flagSet := flag.NewFlagSet("test", 0)
	flagSet.String("http", "true", "test")
	flagSet.String("http.corsdomain", "localhost", "test")
	flagSet.String("http.port", "8889", "test")
	flagSet.String("http.rpcprefix", "testPre", "test")
	flagSet.String("http.vhosts", "*", "test")
	flagSet.String("rpc.batch-request-limit", "1000", "test")
	flagSet.String("rpc.batch-response-max-size", "25000000", "test")

	command := &cli.Command{Name: "mycommand"}

	ctx := cli.NewContext(nil, flagSet, nil)
	ctx.Command = command

	config := getPortalNodeConfig(ctx)

	require.Equal(t, config.ApiHttpEnabled, true)
	require.Equal(t, config.HTTPHost, "127.0.0.1")
	require.Equal(t, config.HTTPPort, 8889)
	require.Equal(t, config.HTTPCors, []string{"localhost"})
	require.Equal(t, config.HTTPVirtualHosts, []string{"*"})
	require.Equal(t, config.HTTPTimeouts, rpc.DefaultHTTPTimeouts)
	require.Equal(t, config.HTTPPathPrefix, "testPre")
	require.Equal(t, config.BatchRequestLimit, 1000)
	require.Equal(t, config.BatchResponseMaxSize, 25000000)

}
