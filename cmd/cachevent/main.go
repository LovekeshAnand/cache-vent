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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cachevent/definitions"
	"cachevent/internal/clean"
	"cachevent/internal/defs"
	"cachevent/internal/envs"
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

Nothing is ever deleted outside quarantine. Quarantined items are purged after 14 days.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmds := map[string]func([]string) error{
		"scan": cmdScan, "clean": cmdClean, "quarantine": cmdList, "restore": cmdRestore, "purge": cmdPurge,
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

// open returns the data dir and quarantine store, purging expired items first.
// There is no background service, so this is when the 14-day purge happens.
func open() (string, quarantine.Store, error) {
	dd, err := defs.DataDir()
	if err != nil {
		return "", quarantine.Store{}, err
	}
	if err := os.MkdirAll(dd, 0o755); err != nil {
		return "", quarantine.Store{}, err
	}
	st := quarantine.Store{DataDir: dd}
	n, b, err := st.Purge(quarantine.MaxAge)
	if n > 0 {
		fmt.Fprintf(os.Stderr, "Purged %d quarantined items older than 14 days (%s).\n", n, human(b))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "purge:", err)
	}
	return dd, st, nil
}

func cmdScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "print JSON (stable format)")
	all := fs.Bool("all", false, "also list items under 10 MB")
	var roots stringList
	fs.Var(&roots, "root", "folder to search for projects and environments (repeatable; default: home)")
	fs.Parse(args)

	entries, err := defs.Load(definitions.FS)
	if err != nil {
		return err
	}
	dd, _, err := open()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if len(roots) == 0 {
		roots = stringList{home}
	}
	fmt.Fprintf(os.Stderr, "Scanning %s ...\n", strings.Join(roots, ", "))
	res := scan.Run(scan.Options{
		Defs:  entries,
		Envs:  envs.Find(),
		Roots: roots,
		// App data is covered by fixed-path definitions; walking it for patterns is slow and noisy.
		Skip: []string{filepath.Join(home, "AppData"), filepath.Join(home, "Library")},
	})
	if err := saveScan(dd, res); err != nil {
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

// saveScan keeps the last scan for clean, and appends per-definition totals to history.
// ponytail: history mixes scans with different --root sets; store roots per line if that skews trends.
func saveScan(dd string, res scan.Result) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dd, "last-scan.json"), b, 0o644); err != nil {
		return err
	}
	sums := map[string]int64{}
	for _, it := range res.Items {
		sums[it.ID] += it.Bytes
	}
	ids := make([]string, 0, len(sums))
	for id := range sums {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	f, err := os.OpenFile(filepath.Join(dd, "history.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, id := range ids {
		if err := enc.Encode(struct {
			Time  time.Time `json:"time"`
			ID    string    `json:"id"`
			Bytes int64     `json:"bytes"`
		}{res.ScannedAt, id, sums[id]}); err != nil {
			return err
		}
	}
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
	entries, err := defs.Load(definitions.FS)
	if err != nil {
		return err
	}
	dd, st, err := open()
	if err != nil {
		return err
	}
	var last scan.Result
	b, err := os.ReadFile(filepath.Join(dd, "last-scan.json"))
	if err != nil {
		return errors.New("no scan yet; run `cachevent scan` first")
	}
	if err := json.Unmarshal(b, &last); err != nil {
		return err
	}
	targets, err := clean.Plan(last, entries, fs.Args(), time.Now())
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
	var moved int64
	var failed []string
	for _, t := range targets {
		m, err := st.Move(t.Path, t.DefinitionID, t.Bytes)
		if err != nil {
			failed = append(failed, err.Error())
			continue
		}
		moved += t.Bytes
		fmt.Printf("moved  %s  (restore: cachevent restore %s)\n", t.Path, m.ID)
	}
	for i, f := range failed {
		if i == 5 {
			fmt.Printf("skipped %d more\n", len(failed)-5)
			break
		}
		fmt.Println("skipped", f)
	}
	fmt.Printf("Moved %s to quarantine. It is purged after 14 days.\n", human(moved))
	return nil
}

func cmdList(args []string) error {
	_, st, err := open()
	if err != nil {
		return err
	}
	items := st.List()
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
	_, st, err := open()
	if err != nil {
		return err
	}
	var errs []error
	for _, id := range args {
		m, err := st.Restore(id)
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
	_, st, err := open() // already purged items past 14 days
	if err != nil || !*all {
		return err
	}
	items := st.List()
	if len(items) == 0 {
		fmt.Println("Quarantine is empty.")
		return nil
	}
	if !*yes && !confirm(fmt.Sprintf("Permanently delete %d quarantined items? This can't be undone.", len(items))) {
		return nil
	}
	n, b, err := st.Purge(0)
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
