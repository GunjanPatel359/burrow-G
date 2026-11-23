// Package commands is every command the command bar knows, one file each.
// Add a command by writing its file here and listing it in New.
package commands

import "github.com/GunjanPatel359/burrow-G/internal/command"

// New returns the command service with every command.
func New(env command.Env) *command.Service {
	var s *command.Service
	help := Help(func() []command.Command { return s.Commands() })
	s = command.NewService(env, Theme, Refresh, help, Quit)
	return s
}
