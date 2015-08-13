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

	for _, c := range containers {
		c.Stop()
	}

	// set up main executors and task generator
	cmdChans := startCurlExecutors(numContainers, startLog())

	for i := 0; i < 19; i++ {
		cmdChans[rand.Intn(numContainers)] <- []byte("Update ")
	}

	time.Sleep(10 * time.Second)
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

func startCurlExecutors(n int, output chan status) []chan command {
	var inChans []chan command
	for i := 0; i < n; i++ {
		inChan := make(chan command)
		go func(i int, in chan command, out chan status) {
			for {
				select {
				case c := <-in:
					if c == nil {
						return
					}

					out <-status(append(c, []byte(fmt.Sprintf("from node %d", i))...))
				}

			}
			// will eventually do the lxc-attach business with curl
			// but just annotating and forwarding for now
		}(i, inChan, output)
		inChans = append(inChans, inChan)
	}
	return inChans
}
