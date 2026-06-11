package merge

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
)

var (
	outlinesRefRe = regexp.MustCompile(`/Outlines\s+(\d+)\s+\d+\s+R`)
	firstRefRe    = regexp.MustCompile(`/First\s+(\d+)\s+\d+\s+R`)
	nextRefRe     = regexp.MustCompile(`/Next\s+(\d+)\s+\d+\s+R`)
)

// collectOutline records the source's outline root and top-level item chain.
// Outline support is best-effort: any parse inconsistency leaves the document
// outline-free instead of failing the merge.
func (d *pdfDocument) collectOutline(rootContent []byte) {
	match := outlinesRefRe.FindSubmatch(rootContent)
	if match == nil {
		return
	}
	rootID, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return
	}
	rootObject, ok := d.objects[rootID]
	if !ok {
		return
	}
	first := firstRefRe.FindSubmatch(rootObject.content)
	if first == nil {
		return
	}
	current, err := strconv.Atoi(string(first[1]))
	if err != nil {
		return
	}

	var tops []int
	visited := make(map[int]bool)
	for current != 0 {
		if visited[current] {
			return // cycle — drop the outline rather than loop forever
		}
		visited[current] = true
		item, ok := d.objects[current]
		if !ok {
			return
		}
		tops = append(tops, current)
		next := nextRefRe.FindSubmatch(item.content)
		if next == nil {
			break
		}
		current, err = strconv.Atoi(string(next[1]))
		if err != nil {
			return
		}
	}
	if len(tops) == 0 {
		return
	}
	d.outlineRootID = rootID
	d.outlineTopIDs = tops
}

// mergedTopItem is one top-level outline entry after object renumbering.
type mergedTopItem struct {
	mergedID  int // renumbered object ID of the item
	source    int // index of the source document
	srcRootID int // the source's outline root (source-local ID)
}

// collectMergedOutlineTops maps every source's top-level outline items
// through objectMap, preserving document order.
func collectMergedOutlineTops(documents []*pdfDocument, objectMap map[objectKey]int) []mergedTopItem {
	var tops []mergedTopItem
	for source, document := range documents {
		if document.outlineRootID == 0 {
			continue
		}
		for _, srcTopID := range document.outlineTopIDs {
			mergedID, ok := objectMap[objectKey{source: source, number: srcTopID}]
			if !ok {
				// inconsistent bookkeeping — drop this source's outline
				tops = tops[:len(tops)-countSource(tops, source)]
				break
			}
			tops = append(tops, mergedTopItem{mergedID: mergedID, source: source, srcRootID: document.outlineRootID})
		}
	}
	return tops
}

func countSource(tops []mergedTopItem, source int) int {
	count := 0
	for _, top := range tops {
		if top.source == source {
			count++
		}
	}
	return count
}

// rewireMergedOutline patches the copied top-level items so they form one
// outline tree under a new merged root object: /Parent is repointed from each
// source's (skipped) root to the merged root, and adjacent items from
// different sources are chained with /Next and /Prev. It writes the merged
// root into copiedObjects under mergedRootID.
//
// The merged root mirrors the engine dialect (internal/pdf putBookmarkRoot):
// /Type /Outlines with /First and /Last only — no /Count.
func rewireMergedOutline(tops []mergedTopItem, copiedObjects map[int][]byte, mergedRootID int) {
	for i, top := range tops {
		content := copiedObjects[top.mergedID]
		// The generic reference rewrite left /Parent pointing at the skipped
		// source root's original number; repoint it at the merged root.
		oldParent := fmt.Sprintf("/Parent %d 0 R", top.srcRootID)
		newParent := fmt.Sprintf("/Parent %d 0 R", mergedRootID)
		content = bytes.Replace(content, []byte(oldParent), []byte(newParent), 1)

		if i > 0 && tops[i-1].source != top.source {
			content = insertBeforeDictEnd(content, fmt.Sprintf("\n/Prev %d 0 R", tops[i-1].mergedID))
		}
		if i+1 < len(tops) && tops[i+1].source != top.source {
			content = insertBeforeDictEnd(content, fmt.Sprintf("\n/Next %d 0 R", tops[i+1].mergedID))
		}
		copiedObjects[top.mergedID] = content
	}

	root := fmt.Sprintf("<</Type /Outlines /First %d 0 R\n/Last %d 0 R>>",
		tops[0].mergedID, tops[len(tops)-1].mergedID)
	copiedObjects[mergedRootID] = []byte(root)
}

// insertBeforeDictEnd inserts addition just before the dictionary's closing
// ">>" delimiter.
func insertBeforeDictEnd(content []byte, addition string) []byte {
	idx := bytes.LastIndex(content, []byte(">>"))
	if idx < 0 {
		return content
	}
	var out bytes.Buffer
	out.Write(content[:idx])
	out.WriteString(addition)
	out.Write(content[idx:])
	return out.Bytes()
}
