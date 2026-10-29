// Command cachevent explains what AI and dev tools store on disk and moves
// what's safe to remove into quarantine.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"cachevent/internal/quarantine"
	"cachevent/internal/scan"
)

const usage = `cachevent: see what AI and dev tools store on disk, and what's safe to remove.

Usage:
  cachevent scan [--json] [--all] [--root DIR]...   find and explain what's on disk
  cachevent clean [--yes] [--dry-run] PATH...       move cleanable items to quarantine
  cachevent quarantine                              list quarantined items
  cachevent restore ID...                           put quarantined items back
  cachevent purge [--all] [--yes]                   delete quarantined items (default: older than 14 days)
  cachevent ui [--no-window]                        open the desktop app

Nothing is ever deleted outside quarantine. Quarantined items are purged after 14 days.
`

// defaultCmd is set to "ui" in the desktop build, so double-clicking it opens the app:
//
//	go build -ldflags "-H windowsgui -X main.defaultCmd=ui" -o CacheVent.exe ./cmd/cachevent
var defaultCmd = ""

func main() {
	if len(os.Args) < 2 {
		if defaultCmd == "ui" {
			os.Args = append(os.Args, "ui")
		} else {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
	}
	cmds := map[string]func([]string) error{
		"scan": cmdScan, "clean": cmdClean, "quarantine": cmdList, "restore": cmdRestore, "purge": cmdPurge, "ui": cmdUI,
	}
	cmd, ok := cmds[os.Args[1]]
	if !ok {
		if a := os.Args[1]; a == "help" || a == "-h" || a == "--help" {
			fmt.Print(usage)
			return
		}
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err := cmd(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "cachevent:", err)
		os.Exit(1)
	}
}

func cmdScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "print JSON (stable format)")
	all := fs.Bool("all", false, "also list items under 10 MB")
	var roots stringList
	fs.Var(&roots, "root", "folder to search for projects and environments (repeatable; default: home)")
	fs.Parse(args)

	s, err := newSession()
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Scanning ...")
	res, err := s.scan(roots, nil)
	if err != nil {
		return err
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	printReport(os.Stdout, res, *all)
	return nil
}

func printReport(w io.Writer, res scan.Result, all bool) {
	var total, cleanable int64
	hidden := 0
	for _, it := range res.Items {
		total += it.Bytes
		if it.Cleanable {
			cleanable += it.Bytes
		}
		if !all && it.Bytes < 10<<20 {
			hidden++
			continue
		}
		mark := " "
		if it.Cleanable {
			mark = "*"
		}
		fmt.Fprintf(w, "%s %9s  %-14s  %-9s  %s\n", mark, human(it.Bytes), it.Safety, ago(it.LastUsed, res.ScannedAt), it.Name)
		const indent = "                                         "
		fmt.Fprintf(w, "%s%s\n", indent, it.Path)
		if e := it.Env; e != nil {
			if e.Project != "" {
				line := "project: " + e.Project
				if e.Orphaned {
					line += "  (MISSING: orphaned)"
				}
				fmt.Fprintf(w, "%s%s\n", indent, line)
			}
			for i, p := range e.Packages {
				if i == 3 || p.Bytes < 1<<20 {
					break
				}
				extra := ""
				if p.OtherEnvs > 0 {
					extra = fmt.Sprintf("  (same version in %d other env(s))", p.OtherEnvs)
				}
				fmt.Fprintf(w, "%s  %s %s  %s%s\n", indent, p.Name, p.Version, human(p.Bytes), extra)
			}
		}
		fmt.Fprintf(w, "%s%s\n\n", indent, it.IfDeleted)
	}
	printSharedPackages(w, res.Items)
	fmt.Fprintf(w, "Found %s, %s of it cleanable (marked *).\n", human(total), human(cleanable))
	if hidden > 0 {
		fmt.Fprintf(w, "%d items under 10 MB hidden; --all shows them.\n", hidden)
	}
	for _, warn := range res.Warnings {
		fmt.Fprintln(w, "warning:", warn)
	}
	if cleanable > 0 {
		fmt.Fprintln(w, "Move items to quarantine with: cachevent clean <path>")
	}
}

// printSharedPackages lists package versions installed in several environments.
func printSharedPackages(w io.Writer, items []scan.Item) {
	type dup struct {
		key   string
		bytes int64
		envs  int
	}
	byKey := map[string]*dup{}
	for _, it := range items {
		if it.Env == nil {
			continue
		}
		for _, p := range it.Env.Packages {
			if p.OtherEnvs == 0 {
				continue
			}
			k := p.Name + " " + p.Version
			if byKey[k] == nil {
				byKey[k] = &dup{key: k}
			}
			byKey[k].bytes += p.Bytes
			byKey[k].envs++
		}
	}
	var dups []*dup
	for _, d := range byKey {
		if waste := d.bytes - d.bytes/int64(d.envs); waste >= 50<<20 {
			dups = append(dups, d)
		}
	}
	if len(dups) == 0 {
		return
	}
	sort.Slice(dups, func(i, j int) bool { return dups[i].bytes > dups[j].bytes })
	fmt.Fprintln(w, "Same package installed in several environments:")
	for i, d := range dups {
		if i == 10 {
			break
		}
		fmt.Fprintf(w, "  %-40s %d envs, %s total\n", d.key, d.envs, human(d.bytes))
	}
	fmt.Fprintln(w, "  uv shares one copy across environments: https://docs.astral.sh/uv/concepts/cache/")
	fmt.Fprintln(w)
}

func cmdClean(args []string) error {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	yes := fs.Bool("yes", false, "don't ask for confirmation")
	dry := fs.Bool("dry-run", false, "show what would be moved")
	fs.Parse(args)
	if fs.NArg() == 0 {
		return errors.New("usage: cachevent clean [--yes] [--dry-run] PATH...")
	}
	s, err := newSession()
	if err != nil {
		return err
	}
	targets, err := s.plan(fs.Args())
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fmt.Println("Nothing old enough to move.")
		return nil
	}
	var total int64
	for _, t := range targets {
		total += t.Bytes
		fmt.Printf("  %9s  %s\n", human(t.Bytes), t.Path)
	}
	if *dry {
		fmt.Printf("Would move %d items (%s) to quarantine.\n", len(targets), human(total))
		return nil
	}
	if !*yes && !confirm(fmt.Sprintf("Move %d items (%s) to quarantine?", len(targets), human(total))) {
		return nil
	}
	r := s.clean(targets)
	for _, m := range r.Moved {
		fmt.Printf("moved  %s  (restore: cachevent restore %s)\n", m.OriginalPath, m.ID)
	}
	failed := r.Failed
	for i, f := range failed {
		if i == 5 {
			fmt.Printf("skipped %d more\n", len(failed)-5)
			break
		}
		fmt.Println("skipped", f)
	}
	fmt.Printf("Moved %s to quarantine. It is purged after 14 days.\n", human(r.Bytes))
	return nil
}

func cmdList(args []string) error {
	s, err := newSession() // purges items past 14 days
	if err != nil {
		return err
	}
	items := s.store.List()
	if len(items) == 0 {
		fmt.Println("Quarantine is empty.")
		return nil
	}
	var total int64
	for _, m := range items {
		total += m.Bytes
		left := time.Until(m.QuarantinedAt.Add(quarantine.MaxAge))
		fmt.Printf("%-32s %9s  purge in %2dd  %s\n", m.ID, human(m.Bytes), int(left.Hours()/24), m.OriginalPath)
	}
	fmt.Printf("%d items, %s.\n", len(items), human(total))
	return nil
}

func cmdRestore(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: cachevent restore ID... (see `cachevent quarantine`)")
	}
	s, err := newSession() // purges items past 14 days
	if err != nil {
		return err
	}
	var errs []error
	for _, id := range args {
		m, err := s.store.Restore(id)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		fmt.Println("restored", m.OriginalPath)
	}
	return errors.Join(errs...)
}

func cmdPurge(args []string) error {
	fs := flag.NewFlagSet("purge", flag.ExitOnError)
	all := fs.Bool("all", false, "purge everything in quarantine now")
	yes := fs.Bool("yes", false, "don't ask for confirmation")
	fs.Parse(args)
	s, err := newSession() // purges items past 14 days
	if err != nil || !*all {
		return err
	}
	items := s.store.List()
	if len(items) == 0 {
		fmt.Println("Quarantine is empty.")
		return nil
	}
	if !*yes && !confirm(fmt.Sprintf("Permanently delete %d quarantined items? This can't be undone.", len(items))) {
		return nil
	}
	n, b, err := s.store.Purge(0)
	fmt.Printf("Purged %d items (%s).\n", n, human(b))
	return err
}

func confirm(q string) bool {
	fmt.Printf("%s [y/N] ", q)
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "y" || s == "yes"
}

func human(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(1024), 0
	for m := n / 1024; m >= 1024; m /= 1024 {
		div *= 1024
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func ago(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	switch d := int(now.Sub(t).Hours() / 24); {
	case d <= 0:
		return "today"
	default:
		return fmt.Sprintf("%dd ago", d)
	}
}

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }
