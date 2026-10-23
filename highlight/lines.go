package highlight

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const GapSentinel = -1

// ParseLineSpec parses a line spec string like "1,21,39,45-51,53-67"
// into a sorted slice of concrete line numbers with -1 gap sentinels inserted
// between non-adjacent ranges.
func ParseLineSpec(s string) ([]int, error) {
	if s == "" {
		return nil, nil
	}

	var nums []int
	for _, seg := range strings.Split(s, ",") {
		seg = strings.TrimSpace(seg)
		if strings.Contains(seg, "-") {
			parts := strings.SplitN(seg, "-", 2)
			lo, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid range %q: %w", seg, err)
			}
			hi, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid range %q: %w", seg, err)
			}
			if lo > hi {
				return nil, fmt.Errorf("invalid range %q: start > end", seg)
			}
			for i := lo; i <= hi; i++ {
				nums = append(nums, i)
			}
		} else {
			n, err := strconv.Atoi(seg)
			if err != nil {
				return nil, fmt.Errorf("invalid line number %q: %w", seg, err)
			}
			nums = append(nums, n)
		}
	}

	// Deduplicate and sort
	seen := map[int]bool{}
	var deduped []int
	for _, n := range nums {
		if !seen[n] {
			seen[n] = true
			deduped = append(deduped, n)
		}
	}
	sort.Ints(deduped)

	// Insert gap sentinels between non-adjacent lines
	var result []int
	for i, n := range deduped {
		if i > 0 && n > deduped[i-1]+1 {
			result = append(result, GapSentinel)
		}
		result = append(result, n)
	}
	return result, nil
}

// BuildLineSpec takes an arbitrary slice of line numbers (may contain duplicates
// or be unsorted) and returns a sorted, deduplicated slice with GapSentinel
// values inserted between non-adjacent lines. Safe to call with nil/empty input.
func BuildLineSpec(nums []int) []int {
	if len(nums) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(nums))
	var deduped []int
	for _, n := range nums {
		if n > 0 && !seen[n] {
			seen[n] = true
			deduped = append(deduped, n)
		}
	}
	sort.Ints(deduped)

	var result []int
	for i, n := range deduped {
		if i > 0 && n > deduped[i-1]+1 {
			result = append(result, GapSentinel)
		}
		result = append(result, n)
	}
	return result
}

// Span is a marked region of one line, in 1-based inclusive columns. The zero
// value marks the whole line — pointing at a 20-character token inside a
// 200-character request body is what makes evidence precise.
type Span struct {
	Start, End int
}

// WholeLine reports whether the span covers the line rather than a column range.
func (s Span) WholeLine() bool { return s.Start == 0 }

// Marks maps a line number to the regions highlighted on it. A line present
// with no spans, or with a whole-line span, is marked end to end.
type Marks map[int][]Span

// Has reports whether a line carries any mark.
func (m Marks) Has(line int) bool { _, ok := m[line]; return ok }

// Lines returns the marked line numbers, sorted.
func (m Marks) Lines() []int {
	out := make([]int, 0, len(m))
	for n := range m {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// ContextSpec returns the line spec covering `context` lines either side of
// every marked line, so a finding can be shown with its surroundings instead of
// the whole file. Returns nil when there is nothing to center on.
//
// It lives here because both commands need it and, for HTTP, the marks are only
// known after tokenizing — a caller that resolved the spec up front could not
// center on a literal found with --mark.
func ContextSpec(m Marks, context int) []int {
	if context <= 0 || len(m) == 0 {
		return nil
	}
	var nums []int
	for _, line := range m.Lines() {
		start := line - context
		if start < 1 {
			start = 1
		}
		for i := start; i <= line+context; i++ {
			nums = append(nums, i)
		}
	}
	return BuildLineSpec(nums)
}

// ParseHighlightSpec parses a highlight spec into the regions to mark.
//
//	13          whole line 13
//	13-15       whole lines 13 through 15
//	13:20-45    line 13, columns 20 through 45
//	13:20       line 13, column 20 only
//	5,13:20-45  both, comma separated
//
// Returns nil for the empty string (nothing highlighted).
func ParseHighlightSpec(s string) (Marks, error) {
	if s == "" {
		return nil, nil
	}

	marks := Marks{}
	for _, seg := range strings.Split(s, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		lineSpec, colSpec, hasCols := strings.Cut(seg, ":")
		if !hasCols {
			nums, err := ParseLineSpec(lineSpec)
			if err != nil {
				return nil, err
			}
			for _, n := range nums {
				if n == GapSentinel {
					continue
				}
				if n < 1 {
					return nil, fmt.Errorf("invalid line in %q: line numbers start at 1", seg)
				}
				marks[n] = append(marks[n], Span{})
			}
			continue
		}

		line, err := strconv.Atoi(strings.TrimSpace(lineSpec))
		if err != nil {
			return nil, fmt.Errorf("invalid line in %q: %w", seg, err)
		}
		if line < 1 {
			return nil, fmt.Errorf("invalid line in %q: line numbers start at 1", seg)
		}
		span, err := parseSpan(seg, colSpec)
		if err != nil {
			return nil, err
		}
		marks[line] = append(marks[line], span)
	}

	if len(marks) == 0 {
		return nil, nil
	}
	return marks, nil
}

// parseSpan reads the "20-45" or "20" half of a line:column spec.
func parseSpan(seg, colSpec string) (Span, error) {
	lo, hi, isRange := strings.Cut(strings.TrimSpace(colSpec), "-")
	start, err := strconv.Atoi(strings.TrimSpace(lo))
	if err != nil {
		return Span{}, fmt.Errorf("invalid column in %q: %w", seg, err)
	}
	end := start
	if isRange {
		end, err = strconv.Atoi(strings.TrimSpace(hi))
		if err != nil {
			return Span{}, fmt.Errorf("invalid column range in %q: %w", seg, err)
		}
	}
	if start < 1 {
		return Span{}, fmt.Errorf("invalid column in %q: columns start at 1", seg)
	}
	if end < start {
		return Span{}, fmt.Errorf("invalid column range in %q: start > end", seg)
	}
	return Span{Start: start, End: end}, nil
}
