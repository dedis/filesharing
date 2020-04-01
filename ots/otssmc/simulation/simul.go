package main

import (
	_ "github.com/calypso-demo/filesharing/ots/service"
	"gopkg.in/dedis/onet.v1/simul"
)

func main() {
	simul.Start()
}
