package util

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
)

// DefaultConfigPath will be used if a --config-path is unset
const DefaultConfigPath = "/etc/config/config.yaml"

// FullNames creates possible full names of a repository
func FullNames(repository scm.Repository) []string {
	owner := repository.Namespace
	name := repository.Name
	fullName := repository.FullName
	if fullName == "" {
		fullName = scm.Join(owner, name)
	}
	fullNames := []string{fullName}
	lowerOwner := strings.ToLower(owner)
	if lowerOwner != owner {
		fullNames = append(fullNames, scm.Join(lowerOwner, name))
	}
	return fullNames
}

// PathOrDefault returns the value for the component's configPath if provided
// explicitly or default otherwise.
func PathOrDefault(value string) string {
	if value != "" {
		return value
	}
	logrus.Warningf("defaulting to %s until 15 July 2019, please migrate", DefaultConfigPath)
	return DefaultConfigPath
}

// DefaultTriggerFor returns the default regexp string used to match comments
// that should trigger the job with this name.
func DefaultTriggerFor(name string) string {
	return fmt.Sprintf(`(?m)^/test( | .* )%s,?($|\s.*)`, name)
}

// DefaultRerunCommandFor returns the default rerun command for the job with
// this name.
func DefaultRerunCommandFor(name string) string {
	return fmt.Sprintf("/test %s", name)
}
