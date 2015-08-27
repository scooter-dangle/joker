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
	cmd chan command
}

func (c *container) Start() bool {
	c.cmd = make(chan command, 25)
	cmd := exec.Command("sudo", "lxc-info", "-n", c.name, "-s")
	info, err := cmd.Output()
	if err != nil {
		log.Fatalf("[error]: %s does not exist. create container and retry.\n", c.name)
		return false
	}
	if strings.Contains(string(info), "RUNNING") {
		log.Printf("[info]: container named %s already running.\n", c.name)
	} else {
		cmd = exec.Command("sudo", "lxc-start", "-n", c.name)
		err = cmd.Run()
		if err != nil {
			log.Fatalf("[error]: could not start %s\n", c.name)
			return false
		}
		cmd = exec.Command("sudo", "lxc-wait", "-n", c.name, "-s", "RUNNING", "-t", "20") // TODO: Make this timeout a package variable of flag.
		err = cmd.Run()
		if err != nil {
			log.Fatalf("[error]: timeout (20 sec) before %s reached RUNNING state.\nRetry with a longer timeout.", c.name)
		}
	}

	c.findIp()
	return true
}

func (c *container) findIp() {
	retriesRemaining := 30 // TODO: should this be a counter or a total timeout?
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
	cmd = exec.Command("sudo", "lxc-wait", "-n", c.name, "-s", "STOPPED", "-t", "20") // TODO: Make this timeout a package variable of flag.
	err = cmd.Run()
	if err != nil {
		log.Printf("[error]: timeout (20 sec) before %s reached STOPPED state.\nRetry with a longer timeout.", c.name)
	}
}

func (c *container) executeCurl(ip string) string {
	cmd := exec.Command("sudo", "lxc-attach", "--clear-env", "-n", c.name, "--", "curl", ip, "-s", "-o", "/dev/null", "-w", "%{http_code}", "-m", "1")
	httpStatus, err := cmd.Output()
	if err != nil {
		log.Printf("[error]: could not attach to %s\n", c.name)
	}
	return string(httpStatus)
}

