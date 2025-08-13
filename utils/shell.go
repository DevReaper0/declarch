package utils

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/fatih/color"
)

func GetApplicationPath(name string) string {
	switch name {
	case "bash":
		return "/bin/bash"
	case "zsh":
		return "/bin/zsh"
	default:
		return name
	}
}

func ExecCommand(command []string, dir string, username string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command provided")
	}
	if command[0] == "" {
		return ExecCommand(command[1:], dir, username)
	}

	color.Set(color.FgCyan)
	fmt.Print("\nRunning command: ")
	color.Set(color.Bold)
	if len(command) > 1 && command[0] == "sh" && command[1] == "-c" {
		fmt.Printf("sh -c %q\n", strings.Join(command[2:], " "))
	} else {
		fmt.Println(strings.Join(command, " "))
	}
	color.Unset()

	cmd := exec.Command(command[0], command[1:]...)
	if dir != "" {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for directory: %w", err)
		}
		cmd.Dir = absDir
	}

	if username != "" && username != "root" {
		userInfo, err := user.Lookup(username)
		if err != nil {
			return fmt.Errorf("failed to get user info for %s: %w", username, err)
		}

		uid, err := strconv.Atoi(userInfo.Uid)
		if err != nil {
			return fmt.Errorf("failed to convert uid to int: %w", err)
		}
		gid, err := strconv.Atoi(userInfo.Gid)
		if err != nil {
			return fmt.Errorf("failed to convert gid to int: %w", err)
		}

		cmd.Env = append(os.Environ(), "USER="+userInfo.Username, "HOME="+userInfo.HomeDir)

		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			},
		}
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}