// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

// Package daemon darwin (mac os x) version
package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"text/template"
)

// darwinRecord - standard record (struct) for darwin version of daemon package
type darwinRecord struct {
	name          string
	description   string
	execStartPath string
	dependencies  []string
}

func newDaemon(name, description, execStartPath string, dependencies []string) (Daemon, error) {

	return &darwinRecord{name, description, execStartPath,dependencies}, nil
}

// Standard service path for system daemons
func (darwin *darwinRecord) servicePath() string {
	return "/Library/LaunchDaemons/" + darwin.name + ".plist"
}

// Is a service installed
func (darwin *darwinRecord) IsInstalled() (bool, error) {
	_, err := os.Stat(darwin.servicePath())
	if err == nil {
		return true, err
	}

	return false, err
}

// Get executable path
func execPath() (string, error) {
	return filepath.Abs(os.Args[0])
}

// Check service is running
func (darwin *darwinRecord) checkRunning() (string, bool) {
	output, err := exec.Command("launchctl", "list", darwin.name).Output()
	if err == nil {
		if matched, err := regexp.MatchString(darwin.name, string(output)); err == nil && matched {
			reg := regexp.MustCompile("PID\" = ([0-9]+);")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				return "Service (pid  " + data[1] + ") is running...", true
			}
			return "Service is running...", true
		}
	}

	return "Service is stopped", false
}

// Install the service
func (darwin *darwinRecord) Install(args ...string) (string, error) {
	installAction := "Install " + darwin.description + ":"

	var err error
	if ok, err := checkPrivileges(); !ok {
		return installAction + failed, err
	}

	srvPath := darwin.servicePath()

	if check, err := darwin.IsInstalled(); check {
		return installAction + failed, err
	}

	if darwin.execStartPath == "" {
		darwin.execStartPath, err = executablePath(darwin.name)
		if err != nil {
			return installAction + failed, err
		}
	}

	if stat, err := os.Stat(darwin.execStartPath); os.IsNotExist(err) || stat.IsDir() {
		return installAction + failed, ErrIncorrectExecStartPath
	}

	file, err := os.Create(srvPath)
	if err != nil {
		return installAction + failed, err
	}
	defer file.Close()

	templ, err := template.New("propertyList").Parse(propertyList)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Path string
			Args       []string
		}{darwin.name, darwin.execStartPath, args},
	); err != nil {
		return installAction + failed, err
	}

	return installAction + success, nil
}

// Remove the service
func (darwin *darwinRecord) Remove() (string, error) {
	removeAction := "Removing " + darwin.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return removeAction + failed, err
	}

	if check, err := darwin.IsInstalled(); !check {
		return removeAction + failed, err
	}

	if err := os.Remove(darwin.servicePath()); err != nil {
		return removeAction + failed, err
	}

	return removeAction + success, nil
}

// Start the service
func (darwin *darwinRecord) Start() (string, error) {
	startAction := "Starting " + darwin.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return startAction + failed, err
	}

	if check, err := darwin.IsInstalled(); !check {
		return startAction + failed, err
	}

	if _, ok := darwin.checkRunning(); ok {
		return startAction + failed, ErrAlreadyRunning
	}

	if err := exec.Command("launchctl", "load", darwin.servicePath()).Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (darwin *darwinRecord) Stop() (string, error) {
	stopAction := "Stopping " + darwin.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return stopAction + failed, err
	}

	if check, err := darwin.IsInstalled(); !check {
		return stopAction + failed, err
	}

	if _, ok := darwin.checkRunning(); !ok {
		return stopAction + failed, ErrAlreadyStopped
	}

	if err := exec.Command("launchctl", "unload", darwin.servicePath()).Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (darwin *darwinRecord) Status() (string, error) {

	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if check, err := darwin.IsInstalled(); !check {
		return "Status could not defined", err
	}

	statusAction, _ := darwin.checkRunning()

	return statusAction, nil
}

// Run - Run service
func (darwin *darwinRecord) Run(e Executable) (string, error) {
	runAction := "Running " + darwin.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

var propertyList = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>KeepAlive</key>
	<true/>
	<key>Label</key>
	<string>{{.Name}}</string>
	<key>ProgramArguments</key>
	<array>
	    <string>{{.Path}}</string>
		{{range .Args}}<string>{{.}}</string>
		{{end}}
	</array>
	<key>RunAtLoad</key>
	<true/>
    <key>WorkingDirectory</key>
    <string>/usr/local/var</string>
    <key>StandardErrorPath</key>
    <string>/usr/local/var/log/{{.Name}}.err</string>
    <key>StandardOutPath</key>
    <string>/usr/local/var/log/{{.Name}}.log</string>
</dict>
</plist>
`
