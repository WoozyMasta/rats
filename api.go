package rats

// Select filters, aggregates, and sorts tags in one call.
// It is equivalent to Sort(Filter(in, opt), opt.Sort, opt.ReleaseOnly).
// When ReleaseOnly is true, Sort will normalize X / X.Y to X.0.0 / X.Y.0 for comparisons.
func Select(in []string, opt Options) []string {
	out := Filter(in, opt)
	if opt.Sort != SortNone {
		out = Sort(out, opt.Sort, opt.ReleaseOnly) // normalize X/X.Y when ReleaseOnly
	}

	return out
}
