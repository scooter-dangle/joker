package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
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
	l := startDisplaySocket()
	defer l.Close()

	containers := launchContainers(numContainers)
	cmdChans := startCurlExecutors(containers, startLog(l))
	curlConnectivityMatrixGenerator(containers, cmdChans)

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

func startDisplaySocket() net.Listener {
	l, err := net.Listen("tcp", "localhost:" + strconv.Itoa(defaultPort))
	if err != nil {
		fmt.Printf("[error]: could not open tcp socket: %v\n", err)
		os.Exit(1)
	}
	return l
}

func curlConnectivityMatrixGenerator(containers []*container, cmdChans []chan command) {
	for {
		for i:= 0; i < len(containers); i++ {
			for j := 0; j < len(containers); j++ {
				cmdChans[i] <- []byte(containers[j].ip)
			}
		}

		time.Sleep(4 * time.Second)
	}
}

func launchContainers(n int) []*container {
	var containers []*container
	for i := 0; i < n; i++ {
		containers = append(containers, &container{name: fmt.Sprintf("n%d", i)})
	}

	for _, c := range containers {
		started := c.Start()
		if !started {
			log.Printf("[error]: failed to start %s\n", c.name)
		}
	}
	return containers
}

func startLog(l net.Listener) chan *status {
	logChan := make(chan *status, 200)
	clientChan := make(chan net.Conn, 10)
	go func() {
		var delay time.Duration
		for {
			client, err := l.Accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					if delay == 0 {
						delay = 5 * time.Millisecond
					} else {
						delay *= 2
					}
					if max := 1 * time.Second; delay > max {
						delay = max
					}
					time.Sleep(delay)
					continue
				}
				return
			}
			delay = 0
			clientChan <- client
		}
	}()

	go func() {
		var clients []net.Conn
		for {
			select {
			case update := <-logChan:
				if update == nil {
					return
				}
				log.Printf("[status]: %s\n", update.summary)
				for _, c := range clients { // TODO: should this be done in goroutines?
					msg := fmt.Sprintf("%s %s %t\n", update.src, update.dest, update.outcome) // TODO: pick a better serialization protocol
					n, err := c.Write([]byte(msg))
					if err != nil {
						continue
					}
					if n != len(msg) {
						log.Println("whole message not sent.")
						// TODO: clearly this should actually try to send the rest of the message...
					}
				}
			case client := <-clientChan:
				clients = append(clients, client) // TODO: clearly not the best data structure (think removes). perhaps a map would be better
			default:
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
