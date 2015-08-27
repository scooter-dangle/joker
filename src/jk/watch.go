package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nsf/termbox-go"
)

func init() {
	cmdWatch.Run = runWatch
}

var cmdWatch = &Command {
	UsageLine: "watch",
	Short:     "watch",
	Long: `
watch displays a dashboard to monitor network connectivity between containers

data for watch comes from daemon

watch supports the following flags:

`,
}

func runWatch(c *Command, args []string) {
	signalChan := make(chan os.Signal, 100)
	signal.Notify(signalChan, syscall.SIGINT)

	startTermbox(signalChan)
	drawGrid()

	go listenForUpdates(signalChan)

	for {
		select {
		case sig := <-signalChan:
			switch sig {
			case syscall.SIGINT:
				termbox.Close()
				os.Exit(0)
			default:
			}
		}

	}
}

func listenForUpdates(signalChan chan os.Signal) {
	retriesRemaining := 4
retry:
	conn, err := net.Dial("tcp", serverIP + ":" + strconv.Itoa(defaultPort)) // eventually this will be udp broadcast autodiscovery...
	if err != nil {
		if retriesRemaining < 0 {
			signalChan <- syscall.SIGINT
			return
		}
		retriesRemaining--
		time.Sleep(600 * time.Millisecond)
		goto retry
	}
	r := bufio.NewReader(conn)
	var to, from string
	var outcome bool
	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			continue
		}
		// parse msg
		// update screen
		_, err = fmt.Sscanf(string(msg), "%s %s %t\n", &from, &to, &outcome)
		if err != nil {
			continue
		}
		drawStatus(status{src: from, dest: to, outcome: outcome})
	}
}

func startTermbox(signalChan chan os.Signal) {
	err := termbox.Init()
	if err != nil {
		log.Fatalf("[error]: could not start termbox: %v\n", err)
	}

	go func() {
		for {
			e := termbox.PollEvent()
			if e.Type == termbox.EventKey {
				if e.Key == termbox.KeyCtrlC {
					signalChan <-syscall.SIGINT
					return
				}
			}
		}
	}()

}

const x0 = 8
const y0 = 2

func drawGrid() {
	// print initial grid and axes
	termbox.SetCell(3, 0, 'f', termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(4, 0, 'r', termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(5, 0, 'o', termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(6, 0, 'm', termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(7, 0, '\\', termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(8, 0, 't', termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(9, 0, 'o', termbox.ColorWhite, termbox.ColorBlack)
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
}

func drawStatus(s status) {
	src, _ := strconv.Atoi(s.src[len(s.src)-1:])
	dest, _ := strconv.Atoi(s.dest[len(s.dest)-1:])
	x, y := curlOutputCell(x0, y0, src, dest)
	fg := termbox.ColorRed
	if s.outcome {
		fg = termbox.ColorGreen
	}

	termbox.SetCell(x, y, '█', fg, termbox.ColorBlack)
	termbox.SetCell(x+1, y, '█', fg, termbox.ColorBlack)
	termbox.Flush()
}

func curlOutputCell(x0, y0, src, dest int) (int, int) {
	x := x0 + 6 + 5*dest
	y := y0 + 1 + 2*src
	return x, y
}
