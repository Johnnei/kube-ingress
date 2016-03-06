package main

import (
	"errors"
	"fmt"
	"os/exec"
)

// Helper function execute commands on the commandline.
func shellOut(cmd string) error {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to execute %v: %v, err: %v", cmd, string(out), err))
	}
	return nil
}

// Helper to merge the name and namespace of a service.
func MergeNameNameSpace(ns, n string) string {
	return ns + "-" + n
}
