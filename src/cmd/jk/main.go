package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nsf/termbox-go"
)

const numContainers = 5

func main() {
	f, err := os.OpenFile("jk.log", os.O_APPEND | os.O_CREATE + os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("[error]: could not open log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	signalChan := make(chan os.Signal, 100)
	signal.Notify(signalChan, syscall.SIGINT)

	// start termbox
	err = termbox.Init()
	if err != nil {
		log.Printf("[error]: could not start termbox: %v\n", err)
	}
	defer termbox.Close()

	go func() {
		for {
			e := termbox.PollEvent()
			if e.Type == termbox.EventKey {
				if e.Key == termbox.KeyCtrlC {
					signalChan <-syscall.SIGINT
					termbox.Close()
					return
				}
			}
		}
	}()

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

func curlOutputCell(x0, y0, src, dest int) (int, int) {
	x := x0 + 6 + 5*dest
	y := y0 + 1 + 2*src
	return x, y
}

func startLog() chan *status {
	logChan := make(chan *status, 200)
	go func() {

		// print initial grid and axes
		termbox.SetCell(3, 0, 'f', termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(4, 0, 'r', termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(5, 0, 'o', termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(6, 0, 'm', termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(7, 0, '\\', termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(8, 0, 't', termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(9, 0, 'o', termbox.ColorWhite, termbox.ColorBlack)
		x0 := 8
		y0 := 2
		termbox.SetCell(x0, y0, '+', termbox.ColorWhite, termbox.ColorBlack)
		for i := 0; i <= numContainers; i++ {
			for j := 0; j <= numContainers; j++ {
				for k := 0; k < 4; k++ {
					termbox.SetCell(x0 + 5*j + k, y0 + 2*i, '-', termbox.ColorWhite, termbox.ColorBlack)
				}
				termbox.SetCell(x0 + 5*j + 4, y0 + (2*i-1), '|', termbox.ColorWhite, termbox.ColorBlack)
				termbox.SetCell(x0 + 5*j + 4, y0 + 2*i, '+', termbox.ColorWhite, termbox.ColorBlack)
			}
		}
		for i := 0; i < numContainers; i++ {
			termbox.SetCell(x0 + 6 + 5*i, y0-1, 'n', termbox.ColorWhite, termbox.ColorBlack)
			termbox.SetCell(x0 + 7 + 5*i, y0-1, rune(i+48), termbox.ColorWhite, termbox.ColorBlack)

			termbox.SetCell(x0 + 1, y0 + 2*i + 1, 'n', termbox.ColorWhite, termbox.ColorBlack)
			termbox.SetCell(x0 + 2, y0 + 2*i + 1, rune(i+48), termbox.ColorWhite, termbox.ColorBlack)
		}
		termbox.HideCursor()
		termbox.Flush()

		for {
			select {
			case update := <-logChan:
				if update == nil {
					return
				}

				log.Printf("[status]: %s\n", update.summary)
				src, _ := strconv.Atoi(update.src[len(update.src)-1:])
				dest, _ := strconv.Atoi(update.dest[len(update.dest)-1:])
				x, y := curlOutputCell(x0, y0, src, dest)
				fg := termbox.ColorRed
				if update.outcome {
					fg = termbox.ColorGreen
				}

				termbox.SetCell(x, y, '█', fg, termbox.ColorBlack)
				termbox.SetCell(x+1, y, '█', fg, termbox.ColorBlack)
				termbox.Flush()
			}
		}
	}()
	return logChan
}

type command []byte
type status struct {
	src string
	dest string
	summary string
	outcome bool
}

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
