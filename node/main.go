package main

import (
	"fmt"
	"log"
	"os"
	"time"

	cli "github.com/urfave/cli/v2"
)

var conf config

func init() {
	conf = config{}
}

func main() {
	app := &cli.NewApp{
		Name:    "P2P-Test-Network",
		Usage:   "Simple p2p grpc Hello message service testing the limits of p2p",
		Flags:   AppConfigFlags,
		Version: "v0.0.1",
		Action:  func(cli *cli.Context) error { return nil },
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Node started at %s and running on %s", time.Now().UTC(), conf.NodeAddr)
}
