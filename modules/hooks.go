package modules

import (
	"fmt"

	"github.com/DevReaper0/declarch/parser"
	"github.com/DevReaper0/declarch/utils"
)

type Hook struct {
	For  string
	When string
	As   string
	Run  string
}

func (h Hook) Exec() error {
	return utils.ExecCommand([]string{"sh", "-c", h.Run}, "", h.As)
}

func HookFrom(section *parser.Section, additionTerm, removalTerm string) (Hook, error) {
	hook := Hook{}

	if forValue := section.GetFirst("for", additionTerm); forValue == additionTerm || forValue == removalTerm {
		hook.For = forValue
	} else {
		return hook, fmt.Errorf("invalid value for 'for' field in hook section: %s (expected '%s' or '%s')", forValue, additionTerm, removalTerm)
	}

	if when := section.GetFirst("when", "after"); when == "before" || when == "after" {
		hook.When = when
	} else {
		return hook, fmt.Errorf("invalid value for 'when' field in hook section: %s", when)
	}

	if as := section.GetFirst("as", PrimaryUser); as != "" {
		hook.As = as
	} else {
		return hook, fmt.Errorf("hook section is missing 'as' field, and no default user is set")
	}

	if run := section.GetFirst("run", ""); run != "" {
		hook.Run = run
	} else {
		return hook, fmt.Errorf("hook section is missing 'run' field")
	}

	return hook, nil
}