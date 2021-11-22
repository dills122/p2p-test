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
	app := &cli.App{
		Flags: AppConfigFlags,
		Name:  "P2P-Test-Network",
		Usage: "Simple p2p grpc Hello message service testing the limits of p2p",
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Node: %s started at %s and running on %s", conf.NodeName, time.Now().UTC(), conf.NodeAddr)
}
