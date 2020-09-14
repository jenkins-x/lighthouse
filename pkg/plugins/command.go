package plugins

import (
	"errors"
	"regexp"
	"strings"

	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
)

// CommandArg defines a plugin command argument
type CommandArg struct {
	Usage    string
	Pattern  string
	Optional bool
}

// GetRegex returns the regex string corresponding to the definiition of a CommandArg
func (a *CommandArg) GetRegex() string {
	pattern := a.Pattern
	if pattern == "" {
		pattern = `[^\r\n]+`
	}
	re := `(?:[ \t]+(` + pattern + "))"
	if a.Optional {
		re += "?"
	}
	return re
}

// GetUsage returns the CommandArg usage
func (a *CommandArg) GetUsage() string {
	usage := a.Usage
	if usage == "" {
		usage = a.Pattern
	}
	if usage == "" {
		usage = "anything"
	}
	if a.Optional {
		return "[" + usage + "]"
	}
	return "<" + usage + ">"
}

// CommandMatch defines a plugin command match to be passed to the command handler
type CommandMatch struct {
	Prefix string
	Name   string
	Arg    string
}

// Command defines a plugin command sent through a comment
type Command struct {
	Prefix      string
	Name        string
	Arg         *CommandArg
	Description string
	Featured    bool
	WhoCanUse   string
	MaxMatches  int
	Filter      func(e scmprovider.GenericCommentEvent) bool
	Handler     CommandEventHandler
	regex       *regexp.Regexp
}

// InvokeCommandHandler performs command checks (filter, then regex if any) the calls the handler with the match (if any)
func (cmd Command) InvokeCommandHandler(ce *scmprovider.GenericCommentEvent, handler func(CommandEventHandler, *scmprovider.GenericCommentEvent, CommandMatch) error) error {
	if cmd.Handler == nil || (cmd.Filter != nil && !cmd.Filter(*ce)) {
		return nil
	}
	regex := cmd.GetRegex()
	if regex != nil {
		max := cmd.MaxMatches
		if max == 0 {
			max = -1
		}
		for _, m := range regex.FindAllStringSubmatch(ce.Body, max) {
			if err := handler(cmd.Handler, ce, cmd.createMatch(m)); err != nil {
				return err
			}
		}
		return nil
	}
	return errors.New("command must have a regexp configured")
}

// GetRegex creates the regular expression from a command syntax
func (cmd *Command) GetRegex() *regexp.Regexp {
	if cmd.regex != nil {
		return cmd.regex
	}
	re := "(?mi)^/(?:lh-)?"
	if cmd.Prefix != "" {
		re += "(" + cmd.Prefix + ")?"
	}
	re += "(" + cmd.Name + ")"
	if cmd.Arg != nil {
		re += cmd.Arg.GetRegex()
	}
	re += `\s*$`
	cmd.regex = regexp.MustCompile(re)
	return cmd.regex
}

// GetMatches returns command matches
func (cmd Command) GetMatches(content string) ([]CommandMatch, error) {
	regex := cmd.GetRegex()
	if regex != nil {
		max := cmd.MaxMatches
		if max == 0 {
			max = -1
		}
		var matches []CommandMatch
		for _, match := range regex.FindAllStringSubmatch(content, max) {
			matches = append(matches, cmd.createMatch(match))
		}
		return matches, nil
	}
	return nil, errors.New("regex cannot be nil")
}

// FilterAndGetMatches filters the event and returns command matches
func (cmd Command) FilterAndGetMatches(event *scmprovider.GenericCommentEvent) ([]CommandMatch, error) {
	if cmd.Handler == nil || (cmd.Filter != nil && !cmd.Filter(*event)) {
		return nil, nil
	}
	return cmd.GetMatches(event.Body)
}

// GetHelp returns command help
func (cmd Command) GetHelp() pluginhelp.Command {
	var examples []string
	for _, name := range strings.Split(cmd.Name, "|") {
		examples = append(examples, "/"+name, "/lh-"+name)
	}
	usage := "/[lh-]"
	if cmd.Prefix != "" {
		usage += "[" + cmd.Prefix + "]"
		for _, name := range strings.Split(cmd.Name, "|") {
			examples = append(examples, "/"+cmd.Prefix+name, "/lh-"+cmd.Prefix+name)
		}
	}
	usage += cmd.Name
	if cmd.Arg != nil {
		usage += " " + cmd.Arg.GetUsage()
		// TODO examples
	}
	who := "Anyone"
	if cmd.WhoCanUse != "" {
		who = cmd.WhoCanUse
	}
	return pluginhelp.Command{
		Usage:       usage,
		Featured:    cmd.Featured,
		Description: cmd.Description,
		Examples:    examples,
		WhoCanUse:   who,
	}
}

// CreateMatch creates a match from an array of strings
func (cmd Command) createMatch(matches []string) CommandMatch {
	match := CommandMatch{}
	if cmd.Prefix != "" {
		match.Prefix = matches[1]
		match.Name = matches[2]
		if cmd.Arg != nil {
			match.Arg = matches[3]
		}
	} else {
		match.Name = matches[1]
		if cmd.Arg != nil {
			match.Arg = matches[2]
		}
	}
	return match
}
