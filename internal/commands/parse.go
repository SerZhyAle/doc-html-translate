package commands

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"doc-html-translate/internal/i18n"
)

// The bounds of APP-COMMANDS rule 3: a request is at most 64 KiB as UTF-8 and one argument at most 32 KiB. More is
// refused, never truncated.
const (
	MaxRequestBytes = 64 << 10
	MaxArgBytes     = 32 << 10
)

// The closed reason tokens of APP-COMMANDS rule 12 that this front can answer with.
const (
	ReasonUnknownVerb = "unknown-verb"
	ReasonBadArgument = "bad-argument"
	ReasonTooLarge    = "too-large"
)

// UsageError is a refusal of the command line (exit 10, rule 3): nothing was executed. Key is the English source
// sentence in internal/i18n and Args its verbs, so the sentence is chosen in the interface language when it is printed.
type UsageError struct {
	Reason string
	Key    string
	Args   []any
}

// Message is the one sentence for a person, in the language lang.
func (e *UsageError) Message(lang string) string { return i18n.T(lang, e.Key, e.Args...) }

// Error is the English sentence, for logs and tests.
func (e *UsageError) Error() string { return e.Message("en") }

func usage(reason, key string, args ...any) *UsageError {
	return &UsageError{Reason: reason, Key: key, Args: args}
}

// Sentences the front prints (each is an internal/i18n entry with 12 translations; i18n_commands.go).
const (
	msgUnknownCommand = `unknown command "%s": "commands" lists the commands`
	msgUnknownOption  = `unknown option "%s" for "%s"`
	msgNeedsValue     = `option "%s" needs a value`
	msgUnexpected     = `unexpected value "%s" for "%s"`
	msgRepeated       = `option "%s" was given more than once`
	msgTooLarge       = `the command line is too large: at most 64 KiB in all and 32 KiB for one argument`
	msgResultFailed   = `the result file could not be written`
	// Added by ticket E0145 (the convert group).
	msgMissing        = `missing required %s for "%s"`
	msgGoogleNeedsCap = `"%s" needs "%s" with a positive amount in USD`
	msgSecretNeedsEnv = `a FileDO secret file needs "%s": a command never asks for a password`
	msgNotServed      = `"%s" is run by doc-html-translate.exe, not by this program`
	msgNoSuchDocument = `the document does not exist`
	// Added by ticket E0147 (the maintenance group).
	msgPackaged         = `this installation is a Microsoft Store package: its file associations come from the package, so "%s" changes nothing here`
	msgOutExists        = `the file named by "%s" already exists: add "%s" to replace it`
	msgOutFolderMissing = `the folder of "%s" does not exist`
	msgNoSpace          = `there is no space left for the archive`
	// Added by ticket E0144 (the launcher first instance).
	msgNotServedByLauncher = `"%s" is run by doc-html-ui.exe, not by this program`
	msgNotRunning          = `doc-html-ui is not running`
	msgLauncherBusy        = `doc-html-ui is busy`
	msgQuitBusy            = `a conversion is running: pass --cancel-runs to stop it and close`
	msgElevationRefused    = `doc-html-ui is running elevated and cannot take commands from a non-elevated caller`
	msgOlderInstance       = `the running copy of doc-html-ui is older and must be restarted`
	msgTimeout             = `the command timed out: doc-html-ui may still be working`
	msgStatusReady         = `doc-html-ui is running`
	msgShowDone            = `the launcher window is open`
	msgQuitDone            = `doc-html-ui closed`
	msgOpenDone            = `queued in doc-html-ui: %d`
	msgPipeFailed          = `the launcher could not complete the command`
)

// CheckBounds refuses a command line over the bounds of rule 3. The request size is the arguments as UTF-8 joined by
// one separator each.
func CheckBounds(args []string) *UsageError {
	total := 0
	for i, a := range args {
		if len(a) > MaxArgBytes {
			return usage(ReasonTooLarge, msgTooLarge)
		}
		total += len(a)
		if i > 0 {
			total++
		}
	}
	if total > MaxRequestBytes {
		return usage(ReasonTooLarge, msgTooLarge)
	}
	return nil
}

var verbShape = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*$`)

// LooksLikeVerb reports whether a token has the shape of a verb word (rule 1) - the shape that a mistyped command has
// and a document path (a dot, a separator, a leading dash) does not.
func LooksLikeVerb(token string) bool { return verbShape.MatchString(token) && utf8.ValidString(token) }

// MatchVerb finds the verb a command line starts with, of one or two words, compared without case, and how many
// tokens it used. The two-word verb is tried first. A line that starts with no verb returns nil.
func (m *Manifest) MatchVerb(args []string) (*Command, int) {
	if len(args) >= 2 && LooksLikeVerb(args[0]) && LooksLikeVerb(args[1]) {
		if c := m.Find(args[0], args[1]); c != nil {
			return c, 2
		}
	}
	if len(args) >= 1 && LooksLikeVerb(args[0]) {
		if c := m.Find(args[0]); c != nil {
			return c, 1
		}
	}
	return nil, 0
}

// hasVerbPrefix reports whether word (compared without case) begins a two-word verb in m. Under APP-COMMANDS 0.2
// item A.1, words that begin a verb of the manifest are reserved first words: when they do not complete a verb,
// the front refuses them (exit 10, unknown command) rather than routing to defaultFor.
func (m *Manifest) hasVerbPrefix(word string) bool {
	if !LooksLikeVerb(word) {
		return false
	}
	for i := range m.Commands {
		if first, _, twoWords := strings.Cut(m.Commands[i].Name, " "); twoWords && strings.EqualFold(first, word) {
			return true
		}
	}
	return false
}

// Invocation is a command line that passed the grammar: the verb and its options, by the canonical (manifest) name of
// each option. A bool option given bare holds "true".
type Invocation struct {
	Command *Command
	Options map[string][]string
}

// Parse reads the tokens after the verb by the grammar of rule 1 and the strictness of rule 3: `--name`, `--name=value`
// or `--name value`, `--` ends the options, options are compared without case and values never. An unknown option, a
// missing value, a value of the wrong kind, a required option or word that is absent, and any word the verb does not
// take are refused.
//
// A verb may take one positional word (the manifest marks it `positional`, ticket E0145): the first token that is not an
// option, or the token after `--`. It is stored under the name of its argument, like an option.
func Parse(c *Command, rest []string) (*Invocation, *UsageError) {
	inv := &Invocation{Command: c, Options: map[string][]string{}}
	pos := c.positional()
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		if tok == "--" {
			// Everything after it is a value, never an option: the positional word, or a refusal.
			for _, w := range rest[i+1:] {
				if err := inv.word(pos, w); err != nil {
					return nil, err
				}
			}
			break
		}
		if !strings.HasPrefix(tok, "--") {
			// A single-dash token is a legacy flag spelling, not an option of a verb; anything else is a word.
			if strings.HasPrefix(tok, "-") && len(tok) > 1 {
				return nil, usage(ReasonBadArgument, msgUnknownOption, tok, c.Name)
			}
			if err := inv.word(pos, tok); err != nil {
				return nil, err
			}
			continue
		}
		name, value, hasValue := strings.Cut(tok[2:], "=")
		arg := c.option(name)
		if arg == nil {
			return nil, usage(ReasonBadArgument, msgUnknownOption, "--"+name, c.Name)
		}
		if arg.Kind == "bool" {
			switch {
			case !hasValue:
				value = "true"
			case strings.EqualFold(value, "true"), strings.EqualFold(value, "false"):
				value = strings.ToLower(value)
			default:
				return nil, usage(ReasonBadArgument, msgUnexpected, value, "--"+name)
			}
		} else if !hasValue {
			if i+1 >= len(rest) {
				return nil, usage(ReasonBadArgument, msgNeedsValue, "--"+name)
			}
			i++
			value = rest[i]
		}
		if err := checkKind(arg, value); err != nil {
			return nil, err
		}
		inv.Options[arg.Name] = append(inv.Options[arg.Name], value)
	}
	for i := range c.Args {
		a := &c.Args[i]
		if a.Required != nil && *a.Required && len(inv.Options[a.Name]) == 0 {
			label := "--" + a.Name
			if a.Positional {
				label = "<" + a.Name + ">"
			}
			return nil, usage(ReasonBadArgument, msgMissing, label, c.Name)
		}
	}
	return inv, nil
}

// positional returns the one positional value of the verb, or nil.
func (c *Command) positional() *Arg {
	for i := range c.Args {
		if c.Args[i].Positional {
			return &c.Args[i]
		}
	}
	return nil
}

// word stores a token that is a value, not an option, as the positional word of the verb. A verb with none, or a second
// word, is refused as an unexpected value.
func (inv *Invocation) word(pos *Arg, w string) *UsageError {
	if pos == nil || len(inv.Options[pos.Name]) > 0 {
		return usage(ReasonBadArgument, msgUnexpected, w, inv.Command.Name)
	}
	if err := checkKind(pos, w); err != nil {
		return err
	}
	inv.Options[pos.Name] = []string{w}
	return nil
}

// option finds a declared option of the verb by its name, without regard to case (APP-COMMANDS 0.2 item E.5: legacy
// option spellings in a.Aliases belong to bare legacy calls, never to canonical verbs). The positional word is not an
// option: `--path` is never accepted for it.
func (c *Command) option(name string) *Arg {
	for i := range c.Args {
		a := &c.Args[i]
		if a.Positional {
			continue
		}
		if strings.EqualFold(a.Name, name) {
			return a
		}
	}
	return nil
}

// checkKind refuses a value of the wrong kind (rule 3). A path must not be empty; an int must be one; an enum must be
// one of its values (compared as written - values are never compared without case).
func checkKind(a *Arg, v string) *UsageError {
	label := "--" + a.Name
	if a.Positional {
		label = "<" + a.Name + ">"
	}
	bad := func() *UsageError { return usage(ReasonBadArgument, msgUnexpected, v, label) }
	switch a.Kind {
	case "path", "string", "guid":
		if v == "" {
			return bad()
		}
	case "int":
		if v == "" {
			return bad()
		}
		for i, r := range v {
			if (r < '0' || r > '9') && !(i == 0 && (r == '-' || r == '+') && len(v) > 1) {
				return bad()
			}
		}
	case "enum":
		for _, ok := range a.Values {
			if v == ok {
				return nil
			}
		}
		return bad()
	}
	return nil
}

// Bool reports a bool option: absent is false, "false" is false, a bare or true option is true.
func (inv *Invocation) Bool(name string) (bool, *UsageError) {
	v, err := inv.single(name)
	if err != nil || v == "" {
		return false, err
	}
	return v == "true", nil
}

// String returns the value of a single-valued option, "" when it was not given.
func (inv *Invocation) String(name string) (string, *UsageError) { return inv.single(name) }

// Strings returns all values of a repeatable option in command-line order.
func (inv *Invocation) Strings(name string) []string {
	vs := inv.Options[name]
	if len(vs) == 0 {
		return nil
	}
	out := make([]string, len(vs))
	copy(out, vs)
	return out
}

func (inv *Invocation) single(name string) (string, *UsageError) {
	vs := inv.Options[name]
	switch len(vs) {
	case 0:
		return "", nil
	case 1:
		return vs[0], nil
	}
	return "", usage(ReasonBadArgument, msgRepeated, "--"+name)
}
