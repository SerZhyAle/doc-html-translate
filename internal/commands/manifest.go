// Package commands is the command front of both executables (APP-COMMANDS 0.1, ticket E0143): the one parser
// that reads a command line by the grammar of rules 1 and 3 before the program does anything else, the manifest
// that says which commands exist (rule 18), and the render of the reference that is generated from it (rule 20).
//
// A call that is not a command of the manifest is not read here at all: it goes to the conversion parser
// (internal/config) untouched, so no call that works today changes meaning.
package commands

import (
	"bytes"
	_ "embed" // the executable's own copy of the manifest
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Product is the product name the manifest, the version line and the reference carry.
const Product = "doc-html-translate"

// embeddedManifest is the executable's own copy of docs/cli-commands.json. go:embed cannot reach docs/, so the copy
// lives here and TestEmbeddedManifestEqualsDocsCopy fails unless the two are byte for byte equal; tools/clidoc
// -render refreshes it (APP-COMMANDS rule 18: "built from the shipped copy, never rewritten from memory").
//
//go:embed cli-commands.json
var embeddedManifest []byte

// Embedded returns the executable's own copy of the manifest, exactly as shipped. Callers must not modify it.
func Embedded() []byte { return embeddedManifest }

// Manifest is the shape of APP-COMMANDS rule 18. A reader ignores a member it does not know, so ParseManifest is lenient;
// Check reads the same bytes strictly and is what the gate runs.
type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	Product       string    `json:"product"`
	Exe           string    `json:"exe"`
	CLIVersion    string    `json:"cliVersion"`
	Commands      []Command `json:"commands"`
}

// Command is one entry of the manifest. The members that rule 18 requires and whose zero value is a real answer
// (`interactive`, `idempotent`, `timeoutMs`, `required`) are pointers, so a missing one is found and not read as false.
type Command struct {
	Name        string   `json:"name"`
	Aliases     []Alias  `json:"aliases"`
	Summary     string   `json:"summary"`
	Synopsis    string   `json:"synopsis"`
	Args        []Arg    `json:"args"`
	Returns     []Return `json:"returns"`
	Risk        string   `json:"risk"`
	ColdStart   string   `json:"coldStart"`
	Wait        string   `json:"wait"`
	TimeoutMs   *int     `json:"timeoutMs"`
	Interactive *bool    `json:"interactive"`
	Idempotent  *bool    `json:"idempotent"`
	Forward     string   `json:"forward,omitempty"`
	// DefaultFor names the executable whose bare call - a path and flags, no verb word - is this verb (ticket E0145).
	// APP-COMMANDS 0.1 has no member for it; a 0.1 reader ignores it (rule 18).
	DefaultFor string          `json:"defaultFor,omitempty"`
	ExitCodes  []ExitCode      `json:"exitCodes"`
	Examples   []Example       `json:"examples"`
	Notes      string          `json:"notes,omitempty"`
	Since      string          `json:"since"`
	Exe        string          `json:"exe,omitempty"`
	Deprecated json.RawMessage `json:"deprecated,omitempty"`
}

// Alias is one pre-adoption spelling that invokes a Command (APP-COMMANDS rule 2 / 0.2 item E.1). In JSON it may be a
// plain string or an object {"spelling": ..., "legacy": true, ...}.
type Alias struct {
	Spelling string            `json:"spelling"`
	Exe      string            `json:"exe,omitempty"`
	Maps     map[string]string `json:"maps,omitempty"`
	Implies  []string          `json:"implies,omitempty"`
	Legacy   bool              `json:"legacy,omitempty"`
	Answers  *bool             `json:"answers,omitempty"`

	fromObject  bool
	hasSpelling bool
}

// UnmarshalJSON accepts either a plain string spelling or the rule-2 object form.
func (a *Alias) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*a = Alias{Spelling: s}
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil || obj == nil {
		return fmt.Errorf("alias must be a string or an object")
	}
	type rawAlias Alias
	var r rawAlias
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	*a = Alias(r)
	a.fromObject = true
	_, a.hasSpelling = obj["spelling"]
	return nil
}

// MarshalJSON writes the object form when any field beyond Spelling (or an object origin) is set, and a plain string otherwise.
func (a Alias) MarshalJSON() ([]byte, error) {
	if !a.fromObject && !a.Legacy && a.Answers == nil && a.Exe == "" && len(a.Maps) == 0 && len(a.Implies) == 0 {
		return json.Marshal(a.Spelling)
	}
	type rawAlias Alias
	return json.Marshal(rawAlias(a))
}

// Arg is one option of a verb (`--name`), or a value it takes.
type Arg struct {
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Required    *bool           `json:"required"`
	Default     json.RawMessage `json:"default,omitempty"`
	Values      []string        `json:"values,omitempty"`
	Aliases     []string        `json:"aliases,omitempty"`
	Description string          `json:"description"`
	// Positional marks the one value a verb takes without an option name (rule 1's `[<value>]`), such as the document of
	// `convert`. It is never written `--name`. APP-COMMANDS 0.1 has no member for it; a 0.1 reader ignores it.
	Positional bool `json:"positional,omitempty"`
}

// Return is one key of the `data` a verb answers with.
type Return struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

// ExitCode is one exit code a verb can end with.
type ExitCode struct {
	Code    int    `json:"code"`
	Meaning string `json:"meaning"`
}

// Example is one worked call: the shell it is written for, the line and what to expect.
type Example struct {
	Shell  string `json:"shell"`
	Line   string `json:"line"`
	Expect string `json:"expect"`
}

// ParseManifest reads a manifest. Unknown members are ignored, as rule 18 asks of a reader.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("commands: manifest is not valid JSON: %w", err)
	}
	return &m, nil
}

// Load reads the executable's own copy.
func Load() (*Manifest, error) { return ParseManifest(embeddedManifest) }

// Find returns the verb with these words (joined by one space, compared without case), or nil.
func (m *Manifest) Find(words ...string) *Command {
	name := strings.ToLower(strings.Join(words, " "))
	for i := range m.Commands {
		if m.Commands[i].Name == name {
			return &m.Commands[i]
		}
	}
	return nil
}

var (
	verbRe    = regexp.MustCompile(`^[a-z][a-z0-9-]*( [a-z][a-z0-9-]*)?$`)
	optionRe  = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	versionRe = regexp.MustCompile(`^\d+\.\d+$`)
)

var (
	riskClasses = map[string]bool{"safe": true, "reads-user-data": true, "writes-user-data": true, "leaves-machine": true}
	coldStarts  = map[string]bool{"start": true, "oneshot": true, "refuse": true}
	waits       = map[string]bool{"done": true, "accepted": true}
	argKinds    = map[string]bool{"string": true, "path": true, "int": true, "bool": true, "enum": true, "guid": true}
	returnKinds = map[string]bool{"string": true, "path": true, "int": true, "bool": true, "enum": true, "guid": true, "object": true, "array": true}
	shells      = map[string]bool{"cmd": true, "powershell": true}
	forwards    = map[string]bool{"": true, "never": true, "always": true}
)

// maxTimeoutMs is the ceiling of rule 8 for a verb that is forwarded.
const maxTimeoutMs = 600000

// Check reads data strictly (a member the structs do not name is a finding, which catches a misspelled member that a
// lenient reader would drop) and returns every way the manifest breaks APP-COMMANDS rule 18 and its closed lists.
// An empty result is a valid manifest.
func Check(data []byte) []string {
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return []string{"manifest does not parse strictly: " + err.Error()}
	}
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	// encoding/json matches member names without regard to case, so "timeoutMS" decodes as "timeoutMs"; a reader in
	// another language (JavaScript, a schema validator) does not, so every member name must be spelt exactly.
	exactMembers(data, reflect.TypeOf(m), "manifest", add)

	if m.SchemaVersion != 1 {
		add("schemaVersion is %d, the shape of rule 18 is 1", m.SchemaVersion)
	}
	if m.Product != Product {
		add("product is %q, want %q", m.Product, Product)
	}
	if m.Exe == "" {
		add("exe is empty")
	}
	if !versionRe.MatchString(m.CLIVersion) {
		add("cliVersion %q is not MAJOR.MINOR", m.CLIVersion)
	}
	if len(m.Commands) == 0 {
		add("commands is empty")
	}

	seenName := map[string]bool{}
	seenAlias := map[string]string{}
	seenDefault := map[string]string{}
	for i := range m.Commands {
		c := &m.Commands[i]
		at := fmt.Sprintf("command %q", c.Name)
		if !verbRe.MatchString(c.Name) {
			add("%s: a verb is one or two lower-case words [a-z][a-z0-9-]* (rule 1)", at)
		}
		if seenName[c.Name] {
			add("%s: listed twice", at)
		}
		seenName[c.Name] = true
		for _, a := range c.Aliases {
			if a.fromObject && !a.hasSpelling {
				add("%s: an alias object needs \"spelling\" (APP-COMMANDS 0.2 item E.1)", at)
				continue
			}
			exe := a.Exe
			if exe == "" {
				exe = c.Exe
			}
			if exe == "" {
				exe = m.Exe
			}
			key := exe + "\x00" + a.Spelling
			if prev, dup := seenAlias[key]; dup {
				add("%s: alias %q is also an alias of %q for %s", at, a.Spelling, prev, exe)
			}
			seenAlias[key] = c.Name
		}
		if c.Aliases == nil {
			add("%s: aliases is missing (an empty array when there is none)", at)
		}
		if c.Summary == "" || c.Synopsis == "" {
			add("%s: summary and synopsis are required", at)
		}
		if !riskClasses[c.Risk] {
			add("%s: risk %q is not one of the four classes of rule 13", at, c.Risk)
		}
		if !coldStarts[c.ColdStart] {
			add("%s: coldStart %q is not start, oneshot or refuse (rule 10)", at, c.ColdStart)
		}
		if !waits[c.Wait] {
			add("%s: wait %q is not done or accepted", at, c.Wait)
		}
		if !forwards[c.Forward] {
			add("%s: forward %q is not never or always", at, c.Forward)
		}
		switch {
		case c.TimeoutMs == nil:
			add("%s: timeoutMs is missing", at)
		case *c.TimeoutMs < 0 || *c.TimeoutMs > maxTimeoutMs:
			add("%s: timeoutMs %d is outside 0..%d (rule 8)", at, *c.TimeoutMs, maxTimeoutMs)
		case *c.TimeoutMs == 0 && c.Forward != "never":
			add("%s: timeoutMs 0 (no bound of its own) is only for a verb that never forwards", at)
		}
		if c.Interactive == nil || c.Idempotent == nil {
			add("%s: interactive and idempotent are required", at)
		}
		if !versionRe.MatchString(c.Since) {
			add("%s: since %q is not MAJOR.MINOR", at, c.Since)
		}
		if len(c.ExitCodes) == 0 {
			add("%s: exitCodes is empty", at)
		}
		for _, e := range c.ExitCodes {
			if e.Code < 0 || e.Meaning == "" {
				add("%s: exit code %d needs a meaning", at, e.Code)
			}
		}
		if len(c.Examples) == 0 {
			add("%s: examples is empty", at)
		}
		for _, e := range c.Examples {
			if !shells[e.Shell] || e.Line == "" || e.Expect == "" {
				add("%s: an example needs shell cmd or powershell, a line and an expect", at)
			}
		}
		if c.Args == nil || c.Returns == nil {
			add("%s: args and returns are required (empty arrays when there are none)", at)
		}
		if c.DefaultFor != "" {
			if prev, dup := seenDefault[c.DefaultFor]; dup {
				add("%s: defaultFor %q is also the default verb of %q (one default verb per executable)", at, c.DefaultFor, prev)
			}
			seenDefault[c.DefaultFor] = c.Name
		}
		seenArg := map[string]bool{}
		positionals := 0
		for _, a := range c.Args {
			if !optionRe.MatchString(a.Name) {
				add("%s: option %q is not --kebab-case (rule 1)", at, a.Name)
			}
			if a.Positional {
				positionals++
				if a.Kind != "path" && a.Kind != "string" {
					add("%s: the positional value %q has kind %q, not path or string", at, a.Name, a.Kind)
				}
			}
			if seenArg[a.Name] {
				add("%s: option %q listed twice", at, a.Name)
			}
			seenArg[a.Name] = true
			if !argKinds[a.Kind] {
				add("%s: option %q has kind %q, not one of string, path, int, bool, enum, guid", at, a.Name, a.Kind)
			}
			if a.Kind == "enum" && len(a.Values) == 0 {
				add("%s: enum option %q lists no values", at, a.Name)
			}
			if a.Required == nil || a.Description == "" {
				add("%s: option %q needs required and a description", at, a.Name)
			}
		}
		if positionals > 1 {
			add("%s: %d positional values, a verb takes at most one", at, positionals)
		}
		for _, r := range c.Returns {
			if r.Name == "" || !returnKinds[r.Kind] || r.Description == "" {
				add("%s: return %q needs a name, a known kind and a description", at, r.Name)
			}
		}
		if m.CLIVersion != "" && versionRe.MatchString(c.Since) && versionRe.MatchString(m.CLIVersion) && versionGreater(c.Since, m.CLIVersion) {
			add("%s: since %s is above the cliVersion %s", at, c.Since, m.CLIVersion)
		}
	}
	// Rule 17: the two stateless base verbs are in every manifest, whatever else the product has.
	for _, base := range []string{"commands", "version"} {
		if !seenName[base] {
			add("the base command %q of rule 17 is missing", base)
		}
	}
	return out
}

// exactMembers reports every object member of raw whose name is not spelt exactly as a json tag of the struct type t
// (descending through slices and nested structs); a json.RawMessage member is opaque.
func exactMembers(raw json.RawMessage, t reflect.Type, path string, add func(string, ...any)) {
	if t == reflect.TypeOf(json.RawMessage{}) {
		return
	}
	switch t.Kind() {
	case reflect.Pointer:
		exactMembers(raw, t.Elem(), path, add)
	case reflect.Slice:
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return
		}
		for i, item := range items {
			exactMembers(item, t.Elem(), fmt.Sprintf("%s[%d]", path, i), add)
		}
	case reflect.Struct:
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return
		}
		fields := map[string]reflect.Type{}
		for i := 0; i < t.NumField(); i++ {
			if name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ","); name != "" {
				fields[name] = t.Field(i).Type
			}
		}
		for key, val := range obj {
			ft, ok := fields[key]
			if !ok {
				add("%s: member %q is not spelt as any member of rule 18 (names are case-sensitive)", path, key)
				continue
			}
			exactMembers(val, ft, path+"."+key, add)
		}
	}
}

// versionGreater compares two MAJOR.MINOR strings numerically.
func versionGreater(a, b string) bool {
	var am, an, bm, bn int
	_, _ = fmt.Sscanf(a, "%d.%d", &am, &an)
	_, _ = fmt.Sscanf(b, "%d.%d", &bm, &bn)
	return am > bm || (am == bm && an > bn)
}
