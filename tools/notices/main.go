// Command notices regenerates static/THIRD_PARTY_NOTICES.txt, the combined
// third-party license notice for the whole product (srvcmanui, srvcman,
// arkgated, subsportal, billportal).
//
// MIT and BSD licenses require their copyright notice and license text to
// accompany binary distributions, so this file has to ship with the OS image
// and be kept in step with the dependencies. Run it from the srvcmanui
// repository root whenever any component's go.mod changes (make notices):
//
//	go run ./tools/notices
//
// It lists every non-own Go module that is compiled into each component for
// OpenBSD/amd64 (go list -deps), reads each module's license file from the Go
// module cache, and adds the front-end libraries bundled in static/ (their
// license texts live in tools/notices/frontend/, one file per library, the
// file name being the heading) and the Go standard library's license.
//
// The component repositories are expected as siblings of this one; override
// with -repos if they live elsewhere. Modules under github.com/rbaylon/ are
// the product's own code and are left out. go list must be able to resolve
// every dependency, so run `go mod download` in a component first if it
// reports a missing module.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const header = `THIRD-PARTY SOFTWARE NOTICES
============================

This product includes third-party software. It is made available under the
licenses reproduced below, which require that these notices accompany
distributions of the software. Inclusion here does not alter the licenses
that apply to the product's own code.

The product consists of the following components:
  srvcmanui   web interface
  srvcman     management API
  arkgated    privileged command daemon
  subsportal  captive portal
  billportal  payment reminder portal

Section 1: Front-end libraries bundled with the web interface (srvcmanui)
Section 2: Go standard library (all components)
Section 3: Go modules compiled into the components
`

const sqliteNote = `
-----
SQLite
-----

github.com/mattn/go-sqlite3 embeds the SQLite database engine, which is in
the public domain (https://www.sqlite.org/copyright.html).
`

const ownPrefix = "github.com/rbaylon/"

var licenseFile = regexp.MustCompile(`(?i)^(licen[sc]e|copying)`)

type module struct {
	versions map[string]string // version -> module cache dir
	usedBy   map[string]bool
}

// text reads a file and normalises its line endings and surrounding space.
func text(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSpace(strings.ReplaceAll(string(b), "\r\n", "\n"))
}

func licenseText(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() && licenseFile.MatchString(e.Name()) {
			return text(filepath.Join(dir, e.Name()))
		}
	}
	log.Fatalf("no license file in %s", dir)
	return ""
}

// collect adds repo's compiled-in third-party modules to mods. The target is
// pinned to the OpenBSD amd64 build the product ships on, with cgo on
// because srvcman and arkgated need it for SQLite.
func collect(name, dir string, mods map[string]*module) {
	cmd := exec.Command("go", "list", "-deps", "-f",
		"{{with .Module}}{{.Path}} {{.Version}} {{.Dir}}{{end}}", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=openbsd", "GOARCH=amd64", "CGO_ENABLED=1")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("go list in %s: %v", dir, err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.SplitN(strings.TrimSpace(line), " ", 3)
		// No version means the main module; no dir means it isn't on disk.
		if len(f) < 3 || f[1] == "" || f[2] == "" || strings.HasPrefix(f[0], ownPrefix) {
			continue
		}
		m := mods[f[0]]
		if m == nil {
			m = &module{versions: map[string]string{}, usedBy: map[string]bool{}}
			mods[f[0]] = m
		}
		m.versions[f[1]] = f[2]
		m.usedBy[name] = true
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func main() {
	repos := flag.String("repos", "., ../srvcman, ../arkgated, ../subsportal, ../billportal",
		"comma-separated component repository directories")
	frontend := flag.String("frontend", "tools/notices/frontend", "directory of front-end license texts")
	out := flag.String("out", "static/THIRD_PARTY_NOTICES.txt", "file to write")
	flag.Parse()

	mods := map[string]*module{}
	for _, r := range strings.Split(*repos, ",") {
		r = strings.TrimSpace(r)
		abs, err := filepath.Abs(r)
		if err != nil {
			log.Fatal(err)
		}
		collect(filepath.Base(abs), abs, mods)
	}

	goroot, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		log.Fatal(err)
	}

	const bar = "=============================================================================="
	var b strings.Builder
	b.WriteString(header + "\n")

	b.WriteString(bar + "\nSECTION 1: FRONT-END LIBRARIES\n" + bar + "\n")
	files, err := filepath.Glob(filepath.Join(*frontend, "*.txt"))
	if err != nil || len(files) == 0 {
		log.Fatalf("no front-end license texts in %s", *frontend)
	}
	sort.Strings(files)
	for _, f := range files {
		title := strings.TrimSuffix(filepath.Base(f), ".txt")
		fmt.Fprintf(&b, "\n-----\n%s\n-----\n\n%s\n\n", title, text(f))
	}

	b.WriteString(bar + "\nSECTION 2: GO STANDARD LIBRARY (BSD-3-Clause)\n" + bar + "\n")
	fmt.Fprintf(&b, "\nThe Go programming language runtime and standard library.\nhttps://go.dev\n\n%s\n\n",
		text(filepath.Join(strings.TrimSpace(string(goroot)), "LICENSE")))

	b.WriteString(bar + "\nSECTION 3: GO MODULES\n" + bar + "\n")
	for _, path := range sortedKeys(mods) {
		m := mods[path]
		vers := sortedKeys(m.versions)
		// The newest version's license text stands for all of them.
		fmt.Fprintf(&b, "\n-----\n%s (%s)\nUsed by: %s\n-----\n\n%s\n\n",
			path, strings.Join(vers, ", "), strings.Join(sortedKeys(m.usedBy), ", "),
			licenseText(m.versions[vers[len(vers)-1]]))
	}
	b.WriteString(sqliteNote)

	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s: %d modules\n", *out, len(mods))
}
