// Package tester checks a 42 project against lightyear's test suites:
// norminette, the Makefile rules, forbidden functions and, for each function
// of the subject, a C suite that runs every case in its own process (so a
// crash or an infinite loop fails only that case) under an allocator that
// catches leaks, double frees, heap overflows and unprotected mallocs.
package tester

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

// Status is the outcome of one test case.
type Status string

// Test outcomes. OK passed; Skip didn't run (a bonus not done, norminette
// not installed); the others failed.
const (
	OK      Status = "OK"
	KO      Status = "KO"
	Crash   Status = "CRASH"
	Timeout Status = "TIMEOUT"
	Skip    Status = "SKIP"
)

// Failed reports whether the case failed.
func (s Status) Failed() bool { return s != OK && s != Skip }

// Case is one test case of a group.
type Case struct {
	Group  string
	Name   string
	Status Status
	// Detail explains a failure (expected vs got, the crash, the leak).
	Detail string
}

// Group is the cases of one function or check, in order.
type Group struct {
	Name  string
	Cases []Case
}

// Failed counts the failed cases.
func (g Group) Failed() int {
	n := 0
	for _, c := range g.Cases {
		if c.Status.Failed() {
			n++
		}
	}
	return n
}

// Skipped reports whether nothing in the group ran.
func (g Group) Skipped() bool {
	for _, c := range g.Cases {
		if c.Status != Skip {
			return false
		}
	}
	return true
}

// Report is the result of a run.
type Report struct {
	Project string
	Cases   []Case
}

// Count returns how many cases passed, failed and were skipped.
func (r Report) Count() (passed, failed, skipped int) {
	for _, c := range r.Cases {
		switch c.Status {
		case OK:
			passed++
		case Skip:
			skipped++
		default:
			failed++
		}
	}
	return passed, failed, skipped
}

// Groups returns the cases grouped, in the order the groups first appear.
func (r Report) Groups() []Group {
	var groups []Group
	index := map[string]int{}
	for _, c := range r.Cases {
		i, ok := index[c.Group]
		if !ok {
			i = len(groups)
			index[c.Group] = i
			groups = append(groups, Group{Name: c.Group})
		}
		groups[i].Cases = append(groups[i].Cases, c)
	}
	return groups
}

// Options tune a run.
type Options struct {
	// CC is the C compiler ("cc" when empty).
	CC string
	// Norminette is the norminette program; empty skips the norm check.
	Norminette string
	// Timeout bounds each test case (5s when zero).
	Timeout time.Duration
	// Only, when set, runs just these functions' suites (the project-wide
	// checks always run).
	Only []string
	// Progress, when set, hears what the run is doing ("make", "compilando
	// os testes"...).
	Progress func(step string)
}

// DefaultOptions uses cc and the norminette on the PATH, if any.
func DefaultOptions() Options {
	opts := Options{CC: "cc"}
	if path, err := lookPath("norminette"); err == nil {
		opts.Norminette = path
	}
	return opts
}

// Project is a 42 project lightyear can test.
type Project struct {
	// Name is what `lightyear test <name>` takes.
	Name string
	// Subject is the project's slug in the subject catalog.
	Subject string
	// Markers are files that must be at the project root.
	Markers []string
	// Machine projects check the computer they run on (born2beroot's VM),
	// not a folder: no copy, and the results print in the terminal.
	Machine bool
	run     func(ctx context.Context, r *runner) error
}

var projects = []Project{
	libft,
	pythonModule("00"), pythonModule("01"), pythonModule("02"), pythonModule("03"), pythonModule("04"),
	born2beroot,
}

// Projects lists the testable projects.
func Projects() []Project { return slices.Clone(projects) }

// Lookup finds a project by name.
func Lookup(name string) (Project, error) {
	for _, p := range projects {
		if p.Name == name {
			return p, nil
		}
	}
	names := make([]string, len(projects))
	for i, p := range projects {
		names[i] = p.Name
	}
	return Project{}, fmt.Errorf("não há testes para %q (disponíveis: %s)", name, strings.Join(names, ", "))
}

// CheckRoot reports an error unless dir looks like the root of the project.
func (p Project) CheckRoot(dir string) error {
	var missing []string
	for _, m := range p.Markers {
		if _, err := os.Stat(filepath.Join(dir, m)); err != nil {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s não parece a raiz do %s: falta %s. Rode o lightyear test na pasta do projeto",
			dir, p.Name, strings.Join(missing, ", "))
	}
	return nil
}

// Run tests the project at dir. Failures of the project are in the report;
// the error is for problems running the tests at all (no compiler, a
// context canceled).
func (p Project) Run(ctx context.Context, dir string, opts Options) (Report, error) {
	if runtime.GOOS == "windows" {
		return Report{}, fmt.Errorf("os testes precisam de Linux ou macOS (usam fork)")
	}
	if err := p.CheckRoot(dir); err != nil {
		return Report{}, err
	}
	if opts.CC == "" {
		opts.CC = "cc"
	}
	r, err := newRunner(dir, opts, !p.Machine)
	if err != nil {
		return Report{}, err
	}
	defer r.close()
	if err := p.run(ctx, r); err != nil {
		return Report{}, err
	}
	return Report{Project: p.Name, Cases: r.cases}, nil
}
