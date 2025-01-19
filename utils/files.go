package utils

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
)

func Chown(path string, username string) error {
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

	err = os.Chown(path, uid, gid)
	if err != nil {
		return err
	}

	return nil
}
