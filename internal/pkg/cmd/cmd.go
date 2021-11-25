package cmd

import "log"

// Command is command-line utility, used for gathering input through prompt from user.
type Command struct {
}

// NewCommand creates a new Command and return its pointer.
func NewCommand() *Command {
	return &Command{}
}

// PromptBool prompt a question with yes/no answer, returns true if user answered yes, returns false otherwise.
func (c *Command) PromptBool(message string) bool {
	// TODO: add implementation
	panic("not implemented")
}

// Log log the message using Command default output.
func (c *Command) Log(message string) {
	log.Println(message)
}
