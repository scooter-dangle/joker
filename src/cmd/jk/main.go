package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

const numContainers = 5

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

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

	for i := 0; i < 19; i++ {
		cmdChans[rand.Intn(numContainers)] <- []byte(containers[rand.Intn(len(containers))].ip)
	}

	time.Sleep(10 * time.Second)

	for _, c := range containers {
		c.Stop()
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
