package main

import (
	"fmt"
	"flag"
	"log"
	"os"
	"github.com/noise/fortune-redis-go/rfortune"
)

var logger = log.New(os.Stdout, "fortune: ", 0)

var (
	redisServer = flag.String("redisServer", ":6379", "host and port for Redis server")
	redisPassword = flag.String("redisPassword", "", "redis password")
)

func main() {
	dir := flag.String("dir", "", "Directory from which to load fortune mods.")
	serve := flag.Bool("serve", false, "Start the HTTP API server.")
	help := flag.Bool("h", false, "Print usage info.")
	flag.Parse()

	rfortune.InitRedis(*redisServer, *redisPassword)

	if *dir != "" {
		rfortune.LoadFortuneMods(*dir)
	} else if *serve {
		rfortune.Start()
	} else if *help {
		flag.Usage()
	} else {
		f := rfortune.RandomFortune("")
		fmt.Println(f.AsText())
	}
}

