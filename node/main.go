package main

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli"
)

var conf config

func init() {
	conf = config{}
}

func main() {
	app := cli.NewApp()
	app.Name = "P2P-Test-Network"
	app.Usage = "Simple p2p grpc Hello message service testing the limits of p2p"
	app.Flags = AppConfigFlags
	app.Version = "v0.0.1"
	app.Action = func(cli *cli.Context) error { return nil }
	app.Run(os.Args)
	fmt.Printf("Node started at %s and running on %s", time.Now().UTC(), conf.NodeAddr)
}
