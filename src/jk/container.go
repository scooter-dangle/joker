package main

import (
	"log"
	"os/exec"
	"strings"
	"time"
)

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

func (c *container) findIp() {
	retriesRemaining := 3
retry:
	cmd := exec.Command("sudo", "lxc-info", "-n", c.name, "-i")
	infoOut, err := cmd.Output()
	infoOutSplit := strings.Split(string(infoOut), ":")
	if (err != nil || len(infoOutSplit) != 2) && retriesRemaining >= 0 {
		retriesRemaining--
		time.Sleep(800 * time.Millisecond)
		goto retry
	}
	if len(infoOutSplit) != 2 {
		log.Println("[error] could not find IP address for %s\n", c.name)
		return
	}
	c.ip = strings.TrimSpace(infoOutSplit[1])
}

func (c *container) Stop() {
	cmd := exec.Command("sudo", "lxc-stop", "-n", c.name)
	err := cmd.Run()
	if err != nil {
		log.Printf("[error]: failed to stop %s\n", c.name)
	}
	// lxc-wait for STOPPED?
}

func (c *container) executeCurl(ip string) string {
	cmd := exec.Command("sudo", "lxc-attach", "-n", c.name, "--", "curl", ip, "-s", "-o", "/dev/null", "-w", "%{http_code}", "-m", "1")
	httpStatus, err := cmd.Output()
	if err != nil {
		log.Printf("[error]: could not attach to %s\n", c.name)
	}
	return string(httpStatus)
}

