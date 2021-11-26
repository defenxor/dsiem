package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Command is command-line utility, used for gathering input through prompt from user.
type Command struct {
	in  io.Reader
	out io.Writer
}

// NewCommand creates a new Command and return its pointer.
func NewCommand(in io.Reader, out io.Writer) *Command {
	return &Command{in, out}
}

// PromptBool prompt a question with yes/no answer, returns true if user answered yes, returns false otherwise.
func (c *Command) PromptBool(message string, def bool) bool {
	var suffix string
	if def {
		suffix = "Y/n"
	} else {
		suffix = "y/N"
	}

	fmt.Fprintf(c.out, "%s (%s): ", message, suffix)
	scanner := bufio.NewScanner(c.in)
	for scanner.Scan() {
		res := scanner.Text()
		res = strings.TrimSpace(strings.ToLower(res))

		if res == "" {
			return def
		}

		if res == "y" || res == "yes" {
			return true
		}

		if res == "n" || res == "no" {
			return false
		}

		fmt.Fprintf(c.out, "please reply with 'y' or 'n': ")
	}

	return def
}

// Log log the message using Command default output.
func (c *Command) Log(message string) {
	fmt.Fprint(c.out, message)
}
