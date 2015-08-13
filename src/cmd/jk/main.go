package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const numContainers = 5

func main() {
	var containers []*container
	for i := 0; i < numContainers; i++ {
		containers = append(containers, &container{name: fmt.Sprintf("n%d", i)})
	}

	for _, c := range containers {
		started := c.Start()
		if !started {
			log.Printf("[error]: failed to start %s\n", c.name)
		}
	}

	for _, c := range containers {
		log.Printf("%s\n", c.ip)
	}

	cmdChans := startCurlExecutors(containers, startLog())

	// curl connectivity matrix generator
	go func(containers []*container, cmdChans []chan command) {
		for {
			for i:= 0; i < len(containers); i++ {
				for j := 0; j < len(containers); j++ {
					cmdChans[i] <- []byte(containers[j].ip)
				}
			}

			time.Sleep(4 * time.Second)
		}
	}(containers, cmdChans)

	signalChan := make(chan os.Signal, 100)
	signal.Notify(signalChan, syscall.SIGINT)
	for {
		select {
		case sig := <-signalChan:
			switch sig {
			case syscall.SIGINT:
				for _, c := range containers {
					c.Stop()
				}
				os.Exit(0)
			default:
			}
		}

	}
}

func startLog() chan status {
	logChan := make(chan status, 200)
	go func() {
		for {
			select {
			case update := <-logChan:
				if update == nil {
					return
				}

				log.Printf("[status]: %s\n", update)
			}
		}

	}()
	return logChan
}

type command []byte
type status []byte

func startCurlExecutors(containers []*container, output chan status) []chan command {
	var inChans []chan command
	for i, c := range containers {
		inChan := make(chan command)
		go func(i int, c *container, in chan command, out chan status) {
			for {
				select {
				case ip := <-in:
					if ip == nil {
						return
					}

					httpStatus := c.executeCurl(string(ip))
					out <-status([]byte(fmt.Sprintf("curl %s from node %d status %s", ip, i, httpStatus)))
				}

			}
		}(i, c, inChan, output)
		inChans = append(inChans, inChan)
	}
	return inChans
}
