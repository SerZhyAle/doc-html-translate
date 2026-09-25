package windowsreg

// Registration is what RegisterHandler achieved, extension by extension. Writing the per-user
// class key is not the same as becoming the handler: since Windows 8 the user's own choice,
// recorded under FileExts\<ext>\UserChoice, wins over it, and only the user may change that.
type Registration struct {
	Default []string // Windows now opens these with the app
	Blocked []string // written, but the user's earlier choice in Windows still wins
	Unknown []string // written, but the user's choice could not be read
	Failed  []string // the registry write failed
}

// Complete reports whether every extension ended up opening with the app.
func (r Registration) Complete() bool {
	return len(r.Blocked) == 0 && len(r.Unknown) == 0 && len(r.Failed) == 0 && len(r.Default) > 0
}

// Status is which handler Windows actually uses for each supported extension.
type Status struct {
	Default []string // Windows opens these with the app
	Blocked []string // registered, but the user's choice in Windows points elsewhere
	Unknown []string // the user's choice could not be read
	Other   []string // neither registered nor chosen
}

// IsDefault reports whether Windows opens every supported extension with the app. An
// unreadable choice counts as "no": a wrong "yes" would hide the step the user has to take.
func (s Status) IsDefault() bool {
	return len(s.Default) > 0 && len(s.Blocked) == 0 && len(s.Unknown) == 0 && len(s.Other) == 0
}

// Summary condenses the status into the one word the GUI switches on: default, blocked,
// unknown, partial or none.
func (s Status) Summary() string {
	switch {
	case s.IsDefault():
		return "default"
	case len(s.Blocked) > 0:
		return "blocked"
	case len(s.Unknown) > 0:
		return "unknown"
	case len(s.Default) > 0:
		return "partial"
	}
	return "none"
}
