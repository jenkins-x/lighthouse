package util

import (
	"strings"
)

const loadFilePrefix = "failed to load file "

// ErrorToMarkdown converts an error to markdown text so we can include it on a Pull Request comment
func ErrorToMarkdown(err error, fileLink string) string {
	lines := strings.Split(err.Error(), ": ")

	// lets convert files to links
	if fileLink != "" {
		for i := range lines {
			line := lines[i]
			if strings.HasPrefix(line, loadFilePrefix) {
				remain := line[len(loadFilePrefix):]
				if remain == "" {
					continue
				}
				words := strings.Split(remain, " ")
				fileName := words[0]
				if len(fileName) == 0 {
					continue
				}
				rest := remain[len(fileName):]
				lines[i] = loadFilePrefix + "[" + fileName + "](" + fileLink + fileName + ")" + rest
			}
		}
	}

	// now lets convert them all to markdown
	return "* " + strings.Join(lines, "\n* ") + "\n"
}
