package fdsec

import (
	"errors"
	"os"

	"doc-html-translate/internal/dialog"
	"doc-html-translate/internal/i18n"
)

// errNoConsole means standard input is not a console, so a no-echo prompt is impossible.
var errNoConsole = errors.New("standard input is not a console")

// consoleAvailable says whether a person can type a password into this process's console. A
// variable so a test does not depend on how go test wires stdin.
var consoleAvailable = stdinIsConsole

// NewAsker chooses where the password comes from, in order: a named environment variable
// (unattended runs), the GUI window that hosts this run (its masked dialog), a console (no-echo
// prompt). With none of them the asker fails at once with a usage error naming the sources - it
// never waits on a question nobody can answer (FDSEC-BEHAVIOUR section 8.1). There is no source
// that puts the password on a command line.
func NewAsker(envName string) Asker {
	switch {
	case envName != "":
		return &envAsker{name: envName}
	case dialog.HostedByGUI():
		return hostAsker{}
	case consoleAvailable():
		return consoleAsker{}
	}
	return noSourceAsker{}
}

// envAsker reads the variable once and keeps the value for the run. Asking again cannot give a
// different answer, so a wrong password ends the run.
type envAsker struct {
	name   string
	value  string
	loaded bool
}

func (a *envAsker) Secret(string, bool) ([]byte, error) {
	if !a.loaded {
		a.value, _ = os.LookupEnv(a.name)
		a.loaded = true
	}
	if a.value == "" {
		return nil, &Error{Class: EmptyEnv, Name: a.name}
	}
	return []byte(a.value), nil
}

func (*envAsker) CanRetry() bool { return false }

// hostAsker puts the question to the GUI that started this run.
type hostAsker struct{}

func (hostAsker) Secret(name string, retry bool) ([]byte, error) {
	msg := i18n.S("Enter the password for %s.", name)
	if retry {
		msg = i18n.S("That password did not open %s - it is wrong, or the file is not a FileDO secret file, or it was altered. Try another password, or cancel.", name)
	}
	secret, ok := dialog.AskSecret(i18n.S("FileDO secret file"), msg, retry)
	if !ok {
		return nil, &Error{Class: Cancelled, Name: name}
	}
	return secret, nil
}

func (hostAsker) CanRetry() bool { return true }

// consoleAsker prompts on the console with echo off, once.
type consoleAsker struct{}

func (consoleAsker) Secret(name string, _ bool) ([]byte, error) {
	secret, err := readConsoleSecret(i18n.S("Password for %s (no echo; an empty password opens only a file made without one - obfuscation, no secrecy): ", name))
	if err != nil {
		if errors.Is(err, errNoConsole) {
			return nil, &Error{Class: NoSource, Name: name}
		}
		return nil, &Error{Class: Cancelled, Name: name}
	}
	return secret, nil
}

func (consoleAsker) CanRetry() bool { return false }

// noSourceAsker is the off-a-terminal case: no variable, no window, no console.
type noSourceAsker struct{}

func (noSourceAsker) Secret(name string, _ bool) ([]byte, error) {
	return nil, &Error{Class: NoSource, Name: name}
}

func (noSourceAsker) CanRetry() bool { return false }
