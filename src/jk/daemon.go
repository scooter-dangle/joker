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

func init() {
	cmdDaemon.Run = runDaemon
}

var cmdDaemon = &Command {
	UsageLine: "daemon",
	Short:     "daemon",
	Long: `
daemon runs a server that manages container lifetimes, executes commands, and sends updates

daemon supports the following flags:

`,
}

func runDaemon(c *Command, args []string) {
	f, err := os.OpenFile("jk.log", os.O_APPEND | os.O_CREATE + os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("[error]: could not open log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	signalChan := make(chan os.Signal, 100)
	signal.Notify(signalChan, syscall.SIGINT)

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

	for {
		select {
		case sig := <-signalChan:
			switch sig {
			case syscall.SIGINT:
				for _, c := range containers {
					c.Stop()
				}
				f.Close()
				os.Exit(0)
			default:
			}
		}

	}
}

func startLog() chan *status {
	logChan := make(chan *status, 200)
	go func() {
		for {
			select {
			case update := <-logChan:
				if update == nil {
					return
				}
				log.Printf("[status]: %s\n", update.summary)
			}
		}
	}()
	return logChan
}

type command []byte

func containerNameByIp(containers []*container, ip string) string {
	for _, c := range containers {
		if c.ip == ip {
			return c.name
		}
	}
	return ""
}


func startCurlExecutors(containers []*container, output chan *status) []chan command {
	var inChans []chan command
	for _, c := range containers {
		inChan := make(chan command)
		go func(c *container, in chan command, out chan *status) {
			for {
				select {
				case ip := <-in:
					if ip == nil {
						return
					}

					success := false
					if "200" == c.executeCurl(string(ip)) {
						success = true
					}

					out <-&status{
						src: c.name,
						dest: containerNameByIp(containers, string(ip)),
						summary: fmt.Sprintf("curl %s from %s %t", ip, c.name, success),
						outcome: success,
					}
				}

			}
		}(c, inChan, output)
		inChans = append(inChans, inChan)
	}
	return inChans
}
