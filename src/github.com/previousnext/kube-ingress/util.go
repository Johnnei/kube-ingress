package main

import (
	"errors"
	"fmt"
	"os"
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

// Helper function to exit the application is errors.
func Check(e error) {
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
}
