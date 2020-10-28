package plugins

import (
	"errors"
	"regexp"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
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

// CommandInvoker defines a plugin command handler and the condition needed for the handler to be invoked
type CommandInvoker struct {
	Condition func(scmprovider.GenericCommentEvent) bool
	Handler   CommandEventHandler
}

// Invoke creates a CommandAction from a handler and a condition
func Invoke(handler CommandEventHandler) CommandInvoker {
	return CommandInvoker{
		Condition: Always,
		Handler:   handler,
	}
}

// Always is a condition that always evaluates to true
func Always(_ scmprovider.GenericCommentEvent) bool {
	return true
}

// Never is a condition that always evaluates to false
func Never(_ scmprovider.GenericCommentEvent) bool {
	return false
}

// Not negates a condition
func Not(condition ConditionFunc) ConditionFunc {
	return func(e scmprovider.GenericCommentEvent) bool {
		return !condition(e)
	}
}

// When creates a CommandAction from an existing CommandAction and a condition (combining conditions with AND)
func (action CommandInvoker) When(conditions ...ConditionFunc) CommandInvoker {
	return CommandInvoker{
		Condition: func(e scmprovider.GenericCommentEvent) bool {
			if !action.Condition(e) {
				return false
			}
			for _, condition := range conditions {
				if !condition(e) {
					return false
				}
			}
			return true
		},
		Handler: action.Handler,
	}
}

// ConditionFunc defines a condition used for invoking commands
type ConditionFunc func(scmprovider.GenericCommentEvent) bool

// Action returns a ConditionFunc that checks event action
func Action(actions ...scm.Action) ConditionFunc {
	return func(e scmprovider.GenericCommentEvent) bool {
		for _, a := range actions {
			if e.Action == a {
				return true
			}
		}
		return false
	}
}

// IsPR returns a ConditionFunc that checks if event concerns a pull request
func IsPR() ConditionFunc {
	return func(e scmprovider.GenericCommentEvent) bool {
		return e.IsPR
	}
}

// IsNotPR returns a ConditionFunc that checks if event doesn't concerns a pull request
func IsNotPR() ConditionFunc {
	return Not(IsPR())
}

// IssueState returns a ConditionFunc that checks event issue state
func IssueState(states ...string) ConditionFunc {
	return func(e scmprovider.GenericCommentEvent) bool {
		for _, s := range states {
			if e.IssueState == s {
				return true
			}
		}
		return false
	}
}

// NotIssueState returns a ConditionFunc that checks event issue state
func NotIssueState(states ...string) ConditionFunc {
	return Not(IssueState(states...))
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
	Action      CommandInvoker
	regex       *regexp.Regexp
}

// InvokeCommandHandler performs command checks (filter, then regex if any) the calls the handler with the match (if any)
func (cmd Command) InvokeCommandHandler(ce *scmprovider.GenericCommentEvent, handler func(CommandEventHandler, *scmprovider.GenericCommentEvent, CommandMatch) error) error {
	if cmd.Action.Handler == nil || (cmd.Action.Condition != nil && !cmd.Action.Condition(*ce)) {
		return nil
	}
	regex := cmd.GetRegex()
	if regex != nil {
		max := cmd.MaxMatches
		if max == 0 {
			max = -1
		}
		for _, m := range regex.FindAllStringSubmatch(ce.Body, max) {
			if err := handler(cmd.Action.Handler, ce, cmd.createMatch(m)); err != nil {
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
	if cmd.Action.Handler == nil || (cmd.Action.Condition != nil && !cmd.Action.Condition(*event)) {
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
