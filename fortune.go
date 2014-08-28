package main

import (
	"flag"
	"fmt"
	"github.com/noise/fortune-redis-go/rfortune"
	"log"
	"os"
)

var (
	logger        = log.New(os.Stdout, "fortune: ", 0)
	redisServer   = flag.String("redisServer", ":6379", "host and port for Redis server")
	redisPassword = flag.String("redisPassword", "", "redis password")
)

func main() {
	help := flag.Bool("h", false, "Print usage info.")
	load := flag.String("load", "", "Load fortune modules from this directory.")
	serve := flag.Bool("serve", false, "Start the HTTP API server.")
	// flags for cmdline fortunes
	verbose := flag.Bool("v", false, "verbose output including fortuneId and module name")
	flag.Parse()

	rfortune.InitRedis(*redisServer, *redisPassword)

	if *load != "" {
		rfortune.LoadFortuneMods(*load)
	} else if *serve {
		rfortune.Start()
	} else if *help {
		flag.Usage()
	} else {
		mod := flag.Arg(0)
		f, err := rfortune.RandomFortune(mod)
		if err != nil {
			logger.Fatalln("|error|", err)
		}
		fmt.Println(f.AsText(*verbose))
	}
}
