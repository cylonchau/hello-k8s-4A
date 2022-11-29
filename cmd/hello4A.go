package main

import (
	"github.com/cylonchau/hello-k8s-4A/server"
)

func main() {
	server.BuildInitFlags()
	server.Run()
}
