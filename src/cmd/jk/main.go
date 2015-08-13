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

func main() {
	ips := startContainers(numContainers)

	for _, ip := range ips {
		fmt.Println(string(ip))
	}

	// set up main executors and task generator
	cmdChans := startCurlExecutors(numContainers, startLog())

	for i := 0; i < 19; i++ {
		cmdChans[rand.Intn(numContainers)] <- []byte("Update ")
	}

	time.Sleep(10 * time.Second)

	stopContainers(numContainers)
}

// Assumes that containers exist and are named n0, n1, ..., nn
func stopContainers(n int) {
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("n%d", i)

		cmd := exec.Command("sudo", "lxc-stop", "-n", name)
		err := cmd.Run()
		if err != nil {
			log.Printf("[error]: failed to stop %s\n", name)
		}
	}
}

// Assumes that containers exist and are named n0, n1, ..., nn
func startContainers(n int) [][]byte {
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("n%d", i)

		// check that container exists and is not running
		cmd := exec.Command("sudo", "lxc-info", "-n", name, "-s")
		info, err := cmd.Output()
		if err != nil {
			log.Fatalf("[error]: %s does not exist. create container and retry.\n", name)
			return nil
		}
		if strings.Contains(string(info), "RUNNING") { // TODO: handle other container states
			log.Printf("[info]: container named %s already running.\n", name)
			continue
		}

		// start container
		cmd = exec.Command("sudo", "lxc-start", "-n", name)
		err = cmd.Run()
		if err != nil {
			log.Fatalf("[error]: could not start %s\n", name)
			return nil
		}
	}

	var ips [][]byte
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("n%d", i)
		ips = append(ips, getIp(name))
	}
	return ips
}

// not perfect, but close enough. allows for things like 442. the goal is not to verify that ip addr provides valid ip addresses though.
var ipRegexp = regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\/24`)
func getIp(name string) []byte {
	retriesRemaining := 3
retry:
	cmd := exec.Command("sudo", "lxc-attach", "-n", name, "--", "ip", "-4", "addr", "show", "eth0")
	ipaddrOut, err := cmd.Output()
	if err != nil {
		log.Fatalf("[error]: could not attach to %s\n", name)
	}
	if len(ipaddrOut) == 0 && retriesRemaining >= 0 {
		retriesRemaining--
		time.Sleep(300 * time.Millisecond)
		goto retry
	}

	match := ipRegexp.FindSubmatch(ipaddrOut)
	if match == nil {
		return nil
	}
	return match[1]
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
