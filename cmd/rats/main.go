package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/woozymasta/rats"
)

type Options struct {
	// SemVer & release behavior
	OptionsSemver OptionsSemver `group:"SemVer and releases"`
	// Output format
	OptionsOutput OptionsOutput `group:"Output"`
	// Aggregation and sorting
	OptionsAggregate OptionsAggregate `group:"Aggregation and sort"`
	// Input filters
	OptionsFilter OptionsFilter `group:"Input filters"`
	// Range clipping
	OptionsRange OptionsRange `group:"Range"`
}

type OptionsSemver struct {
	FilterSemver *bool `short:"s" long:"semver"       description:"Keep only SemVer tags (X.Y.Z[-pre][+build])" default-mask:"true"`
	ReleaseOnly  *bool `short:"r" long:"release-only" description:"Keep only releases (no -pre/+build); allow X / X.Y / X.Y.Z" default-mask:"true"`
	Deduplicate  *bool `short:"d" long:"deduplicate"  description:"Collapse aliases of the same version (MAJOR.MINOR.PATCH+PRERELEASE)" default-mask:"true"`
}

type OptionsOutput struct {
	Canonical bool `short:"c" long:"canonical-out" description:"Print canonical vMAJOR.MINOR.PATCH[-PRERELEASE] (drop +BUILD)"`
	SemVer    bool `short:"v" long:"semver-out"    description:"Print SemVer MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]"`
}

type OptionsAggregate struct {
	Limit         int    `short:"n" long:"limit"    description:"Max number of output tags (<=0 = unlimited)" default:"0"`
	FilterDepth   string `short:"D" long:"depth"    description:"Aggregation depth" choice:"none" choice:"patch" choice:"minor" choice:"major" choice:"latest" default:"minor"`
	SortMode      string `short:"S" long:"sort"     description:"Sort output tags" choice:"none" choice:"asc" choice:"desc" default:"desc"`
	ReleaseFormat string `short:"f" long:"format"   description:"Allowed release forms (when --release-only)" choice:"none" choice:"x" choice:"xy" choice:"xyz" choice:"x-xy" choice:"x-xyz" choice:"xy-xyz" choice:"any" default:"any"`
}

type OptionsFilter struct {
	VPrefixMode string `short:"V" long:"v-prefix"     description:"Policy for leading 'v' in tags" choice:"any" choice:"v" choice:"none" default:"any"`
	Include     string `short:"i" long:"include"      description:"Regexp to keep tags (applied before parsing)"`
	Exclude     string `short:"e" long:"exclude"      description:"Regexp to drop tags (applied before parsing)"`
	ExcludeSigs bool   `short:"E" long:"exclude-sigs" description:"Drop sha256-<64>.sig tags"`
}

type OptionsRange struct {
	Min             string `short:"m" long:"min"                description:"Lower bound (X / X.Y / X.Y.Z or full SemVer)"`
	Max             string `short:"x" long:"max"                description:"Upper bound (X / X.Y / X.Y.Z or full SemVer)"`
	MinExclusive    bool   `short:"M" long:"min-exclusive"      description:"Exclude lower bound itself"`
	MaxExclusive    bool   `short:"X" long:"max-exclusive"      description:"Exclude upper bound itself"`
	IncludePreAtMin bool   `short:"p" long:"include-prerelease" description:"When min is shorthand, include prereleases at the floor (>= X.Y.0-0)"`
}

func main() {
	var opt Options
	parser := flags.NewParser(&opt, flags.Default|flags.AllowBoolValues)
	parser.LongDescription = `RATS — Release App Tag Selector.
A CLI tool for selecting versions from tag lists:
supports SemVer and Go canonical (v-prefixed), can filter prereleases, drop build metadata, sort and aggregate results.`
	if _, err := parser.Parse(); err != nil {
		if flagErr, ok := err.(*flags.Error); ok && flagErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	// Читаем stdin построчно, игнорируем пустые
	in := make([]string, 0, 1024)
	sc := bufio.NewScanner(os.Stdin)
	const maxLine = 10 * 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, maxLine)
	for sc.Scan() {
		if s := strings.TrimSpace(sc.Text()); s != "" {
			in = append(in, s)
		}
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "read stdin: %v", err)
		os.Exit(2)
	}

	if opt.OptionsOutput.Canonical && opt.OptionsOutput.SemVer {
		fmt.Fprintf(os.Stderr, "--canonical-out and --semver-out are mutually exclusive")
		os.Exit(2)
	}

	// Компилим regex (если заданы)
	var incRe, excRe *regexp.Regexp
	if s := strings.TrimSpace(opt.OptionsFilter.Include); s != "" {
		re, err := regexp.Compile(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "include regexp: %v", err)
			os.Exit(2)
		}
		incRe = re
	}
	if s := strings.TrimSpace(opt.OptionsFilter.Exclude); s != "" {
		re, err := regexp.Compile(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "exclude regexp: %v", err)
			os.Exit(2)
		}
		excRe = re
	}

	// Стартуем с дефолтов и переопределяем флагами
	rOpt := rats.DefaultOptions()

	if opt.OptionsSemver.FilterSemver != nil {
		rOpt.FilterSemver = *opt.OptionsSemver.FilterSemver
	}
	if opt.OptionsSemver.ReleaseOnly != nil {
		rOpt.ReleaseOnly = *opt.OptionsSemver.ReleaseOnly
	}
	if opt.OptionsSemver.Deduplicate != nil {
		rOpt.Deduplicate = *opt.OptionsSemver.Deduplicate
	}

	rOpt.ExcludeSignatures = opt.OptionsFilter.ExcludeSigs
	rOpt.VPrefix = rats.ParseVPrefix(opt.OptionsFilter.VPrefixMode)

	rOpt.OutputCanonical = opt.OptionsOutput.Canonical
	rOpt.OutputSemVer = opt.OptionsOutput.SemVer
	rOpt.Include = incRe
	rOpt.Exclude = excRe

	rOpt.Limit = opt.OptionsAggregate.Limit
	rOpt.Depth = rats.ParseDepth(opt.OptionsAggregate.FilterDepth)
	rOpt.Sort = rats.ParseSort(opt.OptionsAggregate.SortMode)
	rOpt.Format = rats.ParseFormat(opt.OptionsAggregate.ReleaseFormat)

	rOpt.Range = rats.Range{
		Min:               strings.TrimSpace(opt.OptionsRange.Min),
		Max:               strings.TrimSpace(opt.OptionsRange.Max),
		MinExclusive:      opt.OptionsRange.MinExclusive,
		MaxExclusive:      opt.OptionsRange.MaxExclusive,
		IncludePrerelease: opt.OptionsRange.IncludePreAtMin,
	}

	out := rats.Select(in, rOpt)
	for _, t := range out {
		fmt.Println(t)
	}
}
