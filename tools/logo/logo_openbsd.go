package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mgutz/ansi"
)

var (
	color = ansi.ColorCode("white+b:green")
	reset = ansi.ColorCode("reset")
)

var logo string = `                                            ` + reset + " Ram:      %s" + color + `
    __________.__  __         .__           ` + reset + " Packages: %d" + color + `
    \______   \__|/  |________|__| ____     ` + reset + " CPU:      %s" + color + `
     |    |  _/  \   __\_  __ \  |/ ___\    ` + reset + " Uptime:   %s" + color + `
     |    |   \  ||  |  |  | \/  / /_/  >   ` + reset + " User:     %s" + color + `
     |______  /__||__|  |__|  |__\___  /    ` + reset + " Hostname: %s" + color + `
            \/                  /_____/     
         Version 1.0                        
                                            `

func getUptime() string {
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		panic(err)
	}

	boottimeint, err := strconv.ParseInt(string(out[:len(out)-1]), 10, 64)
	if err != nil {
		panic(err)
	}

	boottime := time.Unix(boottimeint, 0)
	dur := time.Since(boottime)

	days := int(dur.Hours() / 24)
	hours := int(dur.Hours()) % 24
	minutes := int(dur.Minutes()) % 60

	return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
}

func getPackageCount() int {
	out, err := exec.Command("pkg_info").Output()
	if err != nil {
		panic(err)
	}

	return len(strings.Split(string(out), "\n"))
}

func getCPUName() string {
	out, err := exec.Command("sysctl", "-n", "hw.model").Output()
	if err != nil {
		panic(err)
	}

	return strings.Split(string(out), "@")[0]
}
