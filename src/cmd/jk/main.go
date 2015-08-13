package main

import (
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const numContainers = 5

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type container struct {
	name string
	ip string
}

func (c *container) Start() bool {
	// check that container exists and is not running
	cmd := exec.Command("sudo", "lxc-info", "-n", c.name, "-s")
	info, err := cmd.Output()
	if err != nil {
		log.Fatalf("[error]: %s does not exist. create container and retry.\n", c.name)
		return false
	}
	if strings.Contains(string(info), "RUNNING") { // TODO: handle other container states
		log.Printf("[info]: container named %s already running.\n", c.name)
	} else {
		cmd = exec.Command("sudo", "lxc-start", "-n", c.name)
		err = cmd.Run()
		if err != nil {
			log.Fatalf("[error]: could not start %s\n", c.name)
			return false
		}
	}

	c.findIp()
	return true
}

var ipRegexp = regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\/24`)
// not perfect, but close enough. allows for things like 442.0.0.0 the goal is not to verify that ip addr provides valid ip addresses though.
func (c *container) findIp() {
	retriesRemaining := 3
retry:
	cmd := exec.Command("sudo", "lxc-attach", "-n", c.name, "--", "ip", "-4", "addr", "show", "eth0")
	ipaddrOut, err := cmd.Output()
	if err != nil {
		log.Fatalf("[error]: could not attach to %s\n", c.name)
	}

	match := ipRegexp.FindSubmatch(ipaddrOut)
	if match == nil && retriesRemaining >= 0 {
		retriesRemaining--
		time.Sleep(300 * time.Millisecond)
		goto retry
	}
	c.ip = string(match[1])
}

func (c *container) Stop() {
	cmd := exec.Command("sudo", "lxc-stop", "-n", c.name)
	err := cmd.Run()
	if err != nil {
		log.Printf("[error]: failed to stop %s\n", c.name)
	}
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
