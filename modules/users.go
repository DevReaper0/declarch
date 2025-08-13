package modules

import (
	"fmt"
	"os/user"
	"slices"
	"strconv"
	"strings"

	"github.com/DevReaper0/declarch/parser"
	"github.com/DevReaper0/declarch/utils"
)

type User struct {
	Username   string
	FullName   string
	Shell      string
	CreateHome bool
	HomeDir    string
	Groups     []string
}

func UserFrom(section *parser.Section) (User, error) {
	user := User{}

	if username := section.GetFirst("username", ""); username != "" {
		user.Username = username
	} else {
		return user, fmt.Errorf("user section is missing 'username' field")
	}

	user.FullName = section.GetFirst("full_name", "")
	user.Shell = section.GetFirst("shell", "")

	{
		createHomeString := section.GetFirst("create_home", "true")
		createHome, err := strconv.ParseBool(createHomeString)
		if err != nil {
			return user, fmt.Errorf("invalid value for 'create_home' field in user section '%s': %s", user.Username, createHomeString)
		}
		user.CreateHome = createHome
	}
	user.HomeDir = section.GetFirst("home_dir", "")

	user.Groups = section.GetAll("group")

	return user, nil
}

func CreateUser(user User) error {
	args := []string{"useradd"}

	if user.CreateHome {
		args = append(args, "-m")
	}

	if user.HomeDir != "" {
		args = append(args, "-d", user.HomeDir)
	}

	if user.Shell != "" {
		shellPath := utils.GetApplicationPath(user.Shell)
		args = append(args, "-s", shellPath)
	}

	if user.FullName != "" {
		args = append(args, "-c", user.FullName)
	}

	if len(user.Groups) > 0 {
		args = append(args, "-G", strings.Join(user.Groups, ","))
	}

	args = append(args, user.Username)

	return utils.ExecCommand(args, "", "")
}

func DeleteUser(username string, removeHome bool) error {
	args := []string{"userdel"}

	if removeHome {
		args = append(args, "-r")
	}

	args = append(args, username)

	return utils.ExecCommand(args, "", "")
}

func ModifyUser(previousUser, currentUser User) error {
	args := []string{"usermod"}

	if previousUser.HomeDir != currentUser.HomeDir && currentUser.HomeDir != "" {
		if !currentUser.CreateHome && previousUser.HomeDir != "" {
			args = append(args, "-m")
		}
		args = append(args, "-d", currentUser.HomeDir)
	}

	if previousUser.Shell != currentUser.Shell {
		shellPath := utils.GetApplicationPath(currentUser.Shell)
		args = append(args, "-s", shellPath)
	}

	if previousUser.FullName != currentUser.FullName {
		args = append(args, "-c", currentUser.FullName)
	}

	addedGroups, removedGroups := utils.GetDifferences(currentUser.Groups, previousUser.Groups)
	if len(removedGroups) > 0 {
		userInfo, err := user.Lookup(currentUser.Username)
		if err != nil {
			return fmt.Errorf("failed to get user info for %s: %w", currentUser.Username, err)
		}

		groupIds, err := userInfo.GroupIds()
		if err != nil {
			return fmt.Errorf("failed to get group IDs for user %s: %w", currentUser.Username, err)
		}

		groups := make([]string, 0, len(groupIds)-len(removedGroups)+len(addedGroups))
		for _, groupId := range groupIds {
			group, err := user.LookupGroupId(groupId)
			if err != nil {
				return fmt.Errorf("failed to lookup group ID %s for user %s: %w", groupId, currentUser.Username, err)
			}
			if !slices.Contains(removedGroups, group.Name) {
				groups = append(groups, group.Name)
			}
		}
		groups = append(groups, addedGroups...)

		args = append(args, "-G", strings.Join(groups, ","))
	} else if len(addedGroups) > 0 {
		args = append(args, "-a", "-G", strings.Join(addedGroups, ","))
	}

	args = append(args, currentUser.Username)

	return utils.ExecCommand(args, "", "")
}