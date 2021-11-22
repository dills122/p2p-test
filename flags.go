package main

import (
	cli "github.com/urfave/cli/v2"
)

var AppConfigFlags = []cli.Flag{
        &cli.StringFlag{
            Name:    "name",
            Aliases: []string{"n"},
            Value:   "english",
            Usage:   "The name of the node that will be broadcasted among the network",
			Destination: &conf.NodeName,
        },
        &cli.StringFlag{
            Name:    "listenaddr",
            Aliases: []string{"lad"},
			Value:       "127.0.0.1:10000",
            Usage:   "The address on which the node will begin listening to for requests",
			Destination: &conf.NodeAddr,
        },
        &cli.StringFlag{
            Name:    "service-discover-addr",
            Aliases: []string{"sda"},
			Value:       "127.0.0.1:8500",
            Usage:   "The address used for peer service discovery",
			Destination: &conf.ServiceDiscoveryAddress,
        },
    }

