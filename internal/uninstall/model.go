// Package uninstall removes developer tools, packages and applications together
// with the leftovers they scatter: caches, configuration, shell-profile lines,
// PATH entries and (on Windows) registry keys.
//
// The design keeps one rule: the UI never chooses what runs. Providers describe
// installed things and a Recipe of how to remove each; Inventory.Plan turns a recipe
// into steps the user can review; Execute only runs steps of a stored plan.
// Every path passes safety.Guard, and deletions go through the same deleter as
// the cleaner (Trash by default).
package uninstall

import "bersihdisk/internal/safety"

// Kind groups packages in the UI.
type Kind string

// Kinds of installed thing.
const (
	KindApp     Kind = "app"     // a GUI application
	KindRuntime Kind = "runtime" // a language runtime, version manager or SDK
	KindPackage Kind = "package" // a package installed by a package manager
)

// Installed is one removable thing shown in the uninstall list.
type Installed struct {
	ID       string   `json:"id"`       // provider + ":" + name, stable across listings
	Provider string   `json:"provider"` // npm, pip, cargo, brew, apps, toolchain, ...
	Kind     Kind     `json:"kind"`
	Name     string   `json:"name"`
	Version  string   `json:"version,omitempty"`
	Path     string   `json:"path,omitempty"`  // main install location, when known
	Icon     string   `json:"icon"`            // icon key resolved by the frontend
	Notes    []string `json:"notes,omitempty"` // reason codes shown as hints

	recipe  Recipe
	iconSrc iconSource
}

// Recipe is everything needed to remove one package cleanly. It never leaves
// the backend.
type Recipe struct {
	Commands    []Command     // the native uninstall, run first
	Paths       []Residue     // files and folders to delete afterwards
	Profiles    []ProfileEdit // shell start-up lines to drop
	Registry    []string      // Windows registry keys (exported to a backup, then deleted)
	PathEntries []string      // Windows user PATH entries to drop
	Warnings    []string      // reason codes for the plan header
}

// Command is one process to run. Argv[0] is resolved to an absolute path when
// the plan is built; Raw is a full Windows command line taken from the registry.
type Command struct {
	Argv       []string
	Raw        string
	Label      string
	NeedsAdmin bool
	Optional   bool // shown but not pre-selected
}

// Residue is one location an application leaves behind. Spec may hold ~, {ENV}
// placeholders, an OS prefix and "*" (see Env.Expand).
type Residue struct {
	Spec string
	// Exact = the location is tied to the package by a stable identifier (bundle
	// ID, tool folder). Otherwise it is a name-based guess and starts unchecked.
	Exact bool
	// Reason is a code that makes the step cautionary and unchecked: userData,
	// config, localArtifacts, containerData, nameMatch, ...
	Reason string
	// NeedsAdmin: the location needs administrator rights, so the app only shows
	// the command to run by hand.
	NeedsAdmin bool
}

// StepKind is the type of one plan step.
type StepKind string

// Step kinds, in execution order.
const (
	StepCommand  StepKind = "command"
	StepPath     StepKind = "path"
	StepProfile  StepKind = "profile"
	StepRegistry StepKind = "registry"
	StepEnvPath  StepKind = "envpath"
)

// Step is one reviewable action of a plan.
type Step struct {
	ID       string       `json:"id"`
	Kind     StepKind     `json:"kind"`
	Label    string       `json:"label"`            // command line, path, file, key
	Detail   string       `json:"detail,omitempty"` // e.g. the profile lines to be removed
	Size     int64        `json:"size,omitempty"`
	Level    safety.Level `json:"level"`
	Reasons  []string     `json:"reasons,omitempty"`
	Selected bool         `json:"selected"` // default checkbox state
	// Manual is set when the step needs administrator rights: the app does not
	// run it and shows this command instead.
	Manual string `json:"manual,omitempty"`

	argv    []string
	raw     string
	path    string
	profile profileJob
	key     string
	entries []string
}

// Plan is what uninstalling one package would do.
type Plan struct {
	ID         string    `json:"id"`
	PackageID  string    `json:"packageId"`
	Name       string    `json:"name"`
	Steps      []Step    `json:"steps"`
	Warnings   []Warning `json:"warnings,omitempty"`
	TotalBytes int64     `json:"totalBytes"` // pre-selected path steps
}

// Warning is a caution about the whole plan: a code the frontend translates and
// an optional detail (for example the packages that depend on this one).
type Warning struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

// Progress is one event while a plan runs.
type Progress struct {
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Step  string `json:"step"`
	Bytes int64  `json:"bytes"`
}

// Step outcomes.
const (
	StatusDone    = "done"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
	StatusManual  = "manual"
)

// StepResult is the outcome of one step.
type StepResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	// Hint is a code for a known cause the user can fix (see safety.PermissionHint).
	Hint string `json:"hint,omitempty"`
}

// Result is the outcome of a whole run.
type Result struct {
	OK       int          `json:"ok"`
	Failed   int          `json:"failed"`
	Skipped  int          `json:"skipped"`
	Bytes    int64        `json:"bytes"`
	Steps    []StepResult `json:"steps"`
	Manual   []string     `json:"manual,omitempty"`  // commands to run by hand
	Aborted  bool         `json:"aborted,omitempty"` // a command failed, so nothing after it ran
	Duration float64      `json:"duration"`
}

// Request is what the UI sends to run a plan.
type Request struct {
	PlanID       string   `json:"planId"`
	Steps        []string `json:"steps"`
	Mode         string   `json:"mode"`
	Acknowledged bool     `json:"acknowledged"`
}

// ListResult is the answer to a listing.
type ListResult struct {
	Packages    []Installed `json:"packages"`
	Unavailable []string    `json:"unavailable"` // providers that failed (not "not installed")
}
