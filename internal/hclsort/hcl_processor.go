package hclsort

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// oneNewline is a single-newline separator. Returns a fresh token each
// call: hclwrite.Format mutates spacing in place, and a shared token
// would leak state across insertion points.
func oneNewline() hclwrite.Tokens {
	return hclwrite.Tokens{
		{Type: hclsyntax.TokenNewline, Bytes: []byte{'\n'}},
	}
}

// noSeparator is an empty gap: no padding before the first item or
// after the last.
func noSeparator() hclwrite.Tokens {
	return hclwrite.Tokens{}
}

// ParseHCLContent parses the HCL source byte slice using hclwrite.
func ParseHCLContent(
	src []byte,
	filename string,
) (*hclwrite.File, error) {
	file, diags := hclwrite.ParseConfig(
		src,
		filename,
		hcl.Pos{Line: 1, Column: 1},
	)
	if diags.HasErrors() {
		return nil, fmt.Errorf(
			"error parsing HCL content from '%s': %w",
			filename,
			diags,
		)
	}
	return file, nil
}

// sortItem is an attribute or block being reordered: key is the sort
// key, toks is its own tokens (e.g. from Attribute.BuildTokens).
type sortItem struct {
	key  string
	toks hclwrite.Tokens
}

// reorderBodyAttributes sorts body's attributes by name, preserving any
// token not owned by one (e.g. a comment set off by a blank line).
func reorderBodyAttributes(body *hclwrite.Body) {
	attrs := body.Attributes()
	if len(attrs) == 0 {
		return
	}

	items := make([]sortItem, 0, len(attrs))
	for name, attr := range attrs {
		items = append(items, sortItem{key: name, toks: attr.BuildTokens(nil)})
	}

	// Attached comments can define sections here, unlike in
	// ProcessAndSortBlocks: every attribute in a locals block is equally
	// reorderable, so "# Database" above db_endpoint safely reads as a
	// group label rather than a description of db_endpoint alone.
	rewriteBodyPreservingGaps(body, items, oneNewline(), noSeparator(), noSeparator(), true)
}

// splitLeadComment splits toks' directly attached lead comment (hclwrite
// attaches one when it touches the following item with no blank line)
// from the rest.
func splitLeadComment(toks hclwrite.Tokens) (hclwrite.Tokens, hclwrite.Tokens) {
	idx := 0
	for idx < len(toks) && toks[idx].Type == hclsyntax.TokenComment {
		idx++
	}
	return toks[:idx], toks[idx:]
}

// rewriteBodyPreservingGaps replaces body's contents with items sorted
// by key, preserving every comment currently in body.
//
// HCL attaches a comment to exactly one item, but authors often use one
// to label a whole group (a "section banner"). To avoid reattaching a
// banner to whichever item happens to sort first, items are grouped
// into sections at each comment boundary and sorted only within a
// section; sections keep their source order. A boundary is a floating
// comment, or — if ownCommentIsBoundary — an item's own attached
// comment, which is only safe to treat that way when every item is
// equally reorderable (see ProcessAndSortBlocks for the counterexample).
// A section of one item is just an ordinary attached comment, not a
// group, so it's merged with an adjacent one-item section and stays
// free to reorder against it.
//
// A comment-free gap is replaced by leading/between (tfsort's existing
// spacing convention) rather than preserved verbatim, so reordering
// can't drift blank-line formatting. A gap with a comment is kept, with
// runs of blank lines capped at one.
//
// items' tokens must come from body (e.g. via Body.Attributes() or
// Body.Blocks()): they're matched by *Token identity against
// body.BuildTokens.
func rewriteBodyPreservingGaps(
	body *hclwrite.Body,
	items []sortItem,
	leading, between, trailing hclwrite.Tokens,
	ownCommentIsBoundary bool,
) {
	if len(items) == 0 {
		return
	}

	g := computeItemGaps(body, items, trailing, ownCommentIsBoundary)
	finals := partitionIntoSections(g)

	body.Clear()
	for si, fs := range finals {
		emitSection(body, items, g, fs, si, leading, between)
	}
	if len(g.trailingGap) > 0 {
		body.AppendUnstructuredTokens(g.trailingGap)
	}
}

// itemGaps holds each item's source-order position and the raw gap
// material immediately preceding it, indexed by position in items.
type itemGaps struct {
	selfToks    []hclwrite.Tokens // item's own substantive content
	ownLead     []hclwrite.Tokens // item's own directly attached comment, if split out
	rawPre      []hclwrite.Tokens // true external gap before the item (empty for sourceOrder[0])
	rawLeading  hclwrite.Tokens   // true external gap before sourceOrder[0]
	trailingGap hclwrite.Tokens   // gap after the last item in source order
	sourceOrder []int             // item indices, ordered by position in the source
}

// personalGap is everything that precedes item i and belongs to it
// alone: the true external gap, plus its own attached comment.
func (g *itemGaps) personalGap(i int) hclwrite.Tokens {
	return append(append(hclwrite.Tokens{}, g.rawPre[i]...), g.ownLead[i]...)
}

// computeItemGaps splits each item's attached comment from its content
// (if ownCommentIsBoundary), then computes source-order position and
// preceding gap per item. Position lookups use each item's *original*
// first token, so an item's own comment is never folded into the gap
// before it.
func computeItemGaps(
	body *hclwrite.Body,
	items []sortItem,
	trailing hclwrite.Tokens,
	ownCommentIsBoundary bool,
) *itemGaps {
	g := &itemGaps{
		selfToks: make([]hclwrite.Tokens, len(items)),
		ownLead:  make([]hclwrite.Tokens, len(items)),
		rawPre:   make([]hclwrite.Tokens, len(items)),
	}

	firstTok := make([]*hclwrite.Token, len(items))
	for i, it := range items {
		firstTok[i] = it.toks[0]
		if ownCommentIsBoundary {
			g.ownLead[i], g.selfToks[i] = splitLeadComment(it.toks)
		} else {
			g.selfToks[i] = it.toks
		}
	}

	full := body.BuildTokens(nil)
	pos := make(map[*hclwrite.Token]int, len(full))
	for i, t := range full {
		pos[t] = i
	}

	g.sourceOrder = make([]int, len(items))
	for i := range g.sourceOrder {
		g.sourceOrder[i] = i
	}
	sort.Slice(g.sourceOrder, func(a, b int) bool {
		return pos[firstTok[g.sourceOrder[a]]] < pos[firstTok[g.sourceOrder[b]]]
	})

	cursor := 0
	for n, i := range g.sourceOrder {
		start := pos[firstTok[i]]
		gap := full[cursor:start]
		if n == 0 {
			g.rawLeading = gap
		} else {
			g.rawPre[i] = gap
		}
		cursor = start + len(items[i].toks)
	}
	g.trailingGap = normalizeGap(full[cursor:], trailing)

	return g
}

// finalSection is a contiguous run of same-section items. section0Solo
// marks whether items[0] only leads the section via the solo-merge
// below, rather than genuinely heading a group.
type finalSection struct {
	items        []int
	section0Solo bool
}

// partitionIntoSections groups g's items into sections at comment
// boundaries, then merges any adjacent one-member sections back
// together (see rewriteBodyPreservingGaps).
func partitionIntoSections(g *itemGaps) []finalSection {
	var sections [][]int
	for n, i := range g.sourceOrder {
		if n == 0 || gapHasComment(g.rawPre[i]) || len(g.ownLead[i]) > 0 {
			sections = append(sections, []int{i})
		} else {
			last := len(sections) - 1
			sections[last] = append(sections[last], i)
		}
	}

	wasSolo := make([]bool, len(sections))
	for i, sec := range sections {
		wasSolo[i] = len(sec) == 1
	}

	var finals []finalSection
	for i, sec := range sections {
		if i > 0 && wasSolo[i] && wasSolo[i-1] {
			last := &finals[len(finals)-1]
			last.items = append(last.items, sec...)
			continue
		}
		finals = append(finals, finalSection{items: sec, section0Solo: wasSolo[i]})
	}
	return finals
}

// emitSection sorts fs's items by key and writes them to body with
// their gaps. si == 0 uses the body's true leading gap instead of
// fs.items[0]'s own, since only the first section is body-anchored.
func emitSection(
	body *hclwrite.Body,
	items []sortItem,
	g *itemGaps,
	fs finalSection,
	si int,
	leading, between hclwrite.Tokens,
) {
	section := fs.items
	target := append([]int(nil), section...)
	sort.Slice(target, func(a, b int) bool {
		return items[target[a]].key < items[target[b]].key
	})

	trueLeading := g.rawPre[section[0]]
	if si == 0 {
		trueLeading = g.rawLeading
	}

	displaced := target[0] != section[0]
	// section[0]'s leading gap (trueLeading plus its own comment) is a
	// real banner — fold it into the anchored gap — unless it only
	// leads a solo-merged section and lost the sort to another item;
	// then every bit of it, floating comment included, is personal and
	// must follow its own item instead of staying pinned at the top.
	foldOwnLeadIntoBanner := !displaced || !fs.section0Solo

	var banner hclwrite.Tokens
	if foldOwnLeadIntoBanner {
		banner = append(banner, trueLeading...)
		banner = append(banner, g.ownLead[section[0]]...)
	}
	if displaced {
		banner = append(banner, g.personalGap(target[0])...)
	}
	sectionLeadingGap := normalizeGap(banner, leading)

	if len(sectionLeadingGap) > 0 {
		body.AppendUnstructuredTokens(sectionLeadingGap)
	}
	body.AppendUnstructuredTokens(g.selfToks[target[0]])
	for k := 1; k < len(target); k++ {
		idx := target[k]
		var gapContent hclwrite.Tokens
		if idx == section[0] {
			// Already in the banner above when foldOwnLeadIntoBanner;
			// otherwise none of it — trueLeading included — has been
			// emitted yet, so it all follows the item here instead.
			if !foldOwnLeadIntoBanner {
				gapContent = append(gapContent, trueLeading...)
				gapContent = append(gapContent, g.ownLead[idx]...)
			}
		} else {
			gapContent = append(gapContent, g.rawPre[idx]...)
			gapContent = append(gapContent, g.ownLead[idx]...)
		}
		pre := normalizeGap(gapContent, between)
		if len(pre) > 0 {
			body.AppendUnstructuredTokens(pre)
		}
		body.AppendUnstructuredTokens(g.selfToks[idx])
	}
}

// gapHasComment reports whether gap contains a comment token.
func gapHasComment(gap hclwrite.Tokens) bool {
	for _, t := range gap {
		if t.Type == hclsyntax.TokenComment {
			return true
		}
	}
	return false
}

// normalizeGap collapses runs of blank lines in gap to one, if gap has
// a comment worth keeping. A comment-free gap is pure whitespace, so
// it's replaced by separator instead — keeps reordering from drifting
// blank-line formatting.
func normalizeGap(gap hclwrite.Tokens, separator hclwrite.Tokens) hclwrite.Tokens {
	if !gapHasComment(gap) {
		return cloneTokens(separator) // fresh copy: see cloneTokens
	}

	out := make(hclwrite.Tokens, 0, len(gap))
	blankRun := 0
	for _, t := range gap {
		switch {
		case t.Type == hclsyntax.TokenNewline:
			blankRun++
			if blankRun > 1 {
				continue
			}
		case t.Type == hclsyntax.TokenComment && endsInNewline(t):
			// # and // comments include their own terminating newline.
			blankRun = 0
		default:
			// /* */ comments don't end their own line, so the next
			// newline just terminates the comment, not a blank line.
			blankRun = -1
		}
		out = append(out, t)
	}
	return out
}

// endsInNewline reports whether t's bytes end in a newline: true for #
// and // comments, false for /* */ ones.
func endsInNewline(t *hclwrite.Token) bool {
	return len(t.Bytes) > 0 && t.Bytes[len(t.Bytes)-1] == '\n'
}

// cloneTokens deep-copies toks so the result shares no *Token with it.
// hclwrite.Format mutates a token's spacing in place, so the same
// separator inserted at multiple points must not share token identity.
func cloneTokens(toks hclwrite.Tokens) hclwrite.Tokens {
	if len(toks) == 0 {
		return nil
	}
	out := make(hclwrite.Tokens, len(toks))
	for i, t := range toks {
		cp := *t
		out[i] = &cp
	}
	return out
}

// sortRequiredProvidersInBlock sorts the entries in any required_providers block.
func sortRequiredProvidersInBlock(block *hclwrite.Block) {
	for _, b := range block.Body().Blocks() {
		if b.Type() != "required_providers" {
			continue
		}
		reorderBodyAttributes(b.Body())
	}
}

// sortLocalsBlock sorts the top‐level assignments in a locals block.
func sortLocalsBlock(block *hclwrite.Block) {
	reorderBodyAttributes(block.Body())
}

// ProcessAndSortBlocks extracts sortable blocks (variables, outputs, locals, terraform) and sorts them.
func ProcessAndSortBlocks(
	file *hclwrite.File,
	allowedBlocks map[string]bool,
) *hclwrite.File {
	for _, block := range file.Body().Blocks() {
		switch block.Type() {
		case "terraform":
			sortRequiredProvidersInBlock(block)
		case "locals":
			sortLocalsBlock(block)
		}
	}

	body := file.Body()
	originalBlocks := body.Blocks()

	// Keys encode the existing two-tier order: "\x00"/"\x01" prefix puts
	// every non-sortable block before every sortable one, non-sortable
	// blocks keep original order (zero-padded index), sortable blocks
	// sort by label.
	//
	// Lead comments stay attached here, unlike in reorderBodyAttributes:
	// sections preserve source order, which would conflict with this
	// key scheme's global reordering — stripping could let one comment
	// carve a sortable block out of its "always after non-sortable"
	// slot. A `/* */` banner is still caught, since hclwrite never
	// attaches those; only a directly-touching `#`/`//` comment on a
	// sortable block goes unrecognized here.
	items := make([]sortItem, 0, len(originalBlocks))
	otherIdx := 0
	for _, block := range originalBlocks {
		if allowedBlocks[block.Type()] && len(block.Labels()) > 0 {
			items = append(items, sortItem{
				key:  "\x01" + block.Labels()[0],
				toks: block.BuildTokens(nil),
			})
			continue
		}
		items = append(items, sortItem{
			key:  fmt.Sprintf("\x00%08d", otherIdx),
			toks: block.BuildTokens(nil),
		})
		otherIdx++
	}

	rewriteBodyPreservingGaps(body, items, noSeparator(), oneNewline(), noSeparator(), false)

	return file
}

// FormatHCLBytes formats the HCL file's content into a byte slice.
func FormatHCLBytes(file *hclwrite.File) []byte {
	return hclwrite.Format(file.Bytes())
}
