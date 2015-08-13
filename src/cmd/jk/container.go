package main

import (
	"log"
	"os/exec"
	"regexp"
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

var ipRegexp = regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\/24`)
// not perfect, but close enough. allows for things like 442.0.0.0 the goal is not to verify that ip addr provides valid ip addresses though.
func (c *container) findIp() {
	retriesRemaining := 3
retry:
	cmd := exec.Command("sudo", "lxc-attach", "-n", c.name, "--", "ip", "-4", "addr", "show", "eth0")
	ipaddrOut, err := cmd.Output()
	match := ipRegexp.FindSubmatch(ipaddrOut)
	if (err != nil || match == nil) && retriesRemaining >= 0 {
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

func (c *container) executeCurl(ip string) string {
	cmd := exec.Command("sudo", "lxc-attach", "-n", c.name, "--", "curl", ip, "-s", "-o", "/dev/null", "-w", "%{http_code}", "-m", "1")
	httpStatus, err := cmd.Output()
	if err != nil {
		log.Fatalf("[error]: could not attach to %s\n", c.name)
	}
	return string(httpStatus)
}

