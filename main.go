package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type PodmanConnection struct {
	Name     string `json:"Name"`
	URI      string `json:"URI"`
	Identity string `json:"Identity"`
	Default  bool
}

func getPodManConnections() []PodmanConnection {
	cmd := exec.Command("podman", "system", "connection", "list", "--format=json")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	var connections []PodmanConnection
	json.Unmarshal(out.Bytes(), &connections)
	for i, connection := range connections {
		connections[i].Default = strings.HasSuffix(connection.Name, "*")
		connections[i].Name = strings.TrimSuffix(connection.Name, "*")
	}
	return connections
}

func getDefaultMachineName(connections []PodmanConnection) string {
	for _, connection := range connections {
		if connection.Default {
			return connection.Name
		}
	}
	panic("no default machine")
}

func disableSeLinux(machineName string) {
	cmd := exec.Command("podman", "machine", "ssh", machineName, "sudo", "setenforce", "0")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	cmd = exec.Command("podman", "machine", "ssh", machineName, "sudo", "sestatus")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func findPodmanConnection(connections []PodmanConnection, machineName string) PodmanConnection {
	for _, connection := range connections {
		if connection.Name == machineName {
			return connection
		}
	}
	panic("no matched connection")
}

func findRootPodmanConnection(connections []PodmanConnection, machineName string) PodmanConnection {
	rootConnectionName := machineName + "-root"
	for _, connection := range connections {
		if connection.Name == rootConnectionName {
			return connection
		}
	}
	panic("no matched connection")
}

func sshPortForwarding(connection PodmanConnection) {
	u, err := url.Parse(connection.URI)
	if err != nil {
		log.Fatal(err)
	}
	dockerHostname := fmt.Sprintf("%s@%s", u.User, u.Hostname())
	portForwarding := fmt.Sprintf("127.0.0.1:2375:%s", u.Path)
	cmd := exec.Command("ssh", "-i", connection.Identity, "-o", "StrictHostKeyChecking=no", "-L",
		portForwarding, "-N", "-p", u.Port(), dockerHostname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	connections := getPodManConnections()
	machineName := getDefaultMachineName(connections)
	disableSeLinux(machineName)
	connection := findRootPodmanConnection(connections, machineName)
	sshPortForwarding(connection)
}
