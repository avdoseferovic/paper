package merge

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

const (
	mergedCatalogObjectID = 1
	mergedPagesObjectID   = 2
	firstCopiedObjectID   = 3
)

var parentRefRe = regexp.MustCompile(`/Parent\s+\d+\s+\d+\s+R`)

type objectKey struct {
	source int
	number int
}

func writeMergedPDF(documents []*pdfDocument) ([]byte, error) {
	objectMap := make(map[objectKey]int)
	nextObjectID := firstCopiedObjectID
	for source, document := range documents {
		for _, objectID := range sortedObjectIDs(document.objects) {
			if shouldSkipObject(document, objectID) {
				continue
			}
			objectMap[objectKey{source: source, number: objectID}] = nextObjectID
			nextObjectID++
		}
	}

	pageIDs := make([]int, 0)
	copiedObjects := make(map[int][]byte, nextObjectID-firstCopiedObjectID)
	for source, document := range documents {
		pageSet := make(map[int]struct{}, len(document.pageIDs))
		for _, pageID := range document.pageIDs {
			pageSet[pageID] = struct{}{}
			pageIDs = append(pageIDs, objectMap[objectKey{source: source, number: pageID}])
		}

		for _, objectID := range sortedObjectIDs(document.objects) {
			if shouldSkipObject(document, objectID) {
				continue
			}
			newObjectID := objectMap[objectKey{source: source, number: objectID}]
			content := rewriteObjectReferences(document.objects[objectID].content, source, objectMap)
			if _, ok := pageSet[objectID]; ok {
				content = replacePageParent(content, mergedPagesObjectID)
			}
			copiedObjects[newObjectID] = content
		}
	}

	if len(pageIDs) == 0 {
		return nil, fmt.Errorf("%w: no pages to merge", errUnsupportedPDF)
	}

	return renderMergedPDF(maxPDFVersion(documents), pageIDs, copiedObjects, nextObjectID), nil
}

func shouldSkipObject(document *pdfDocument, objectID int) bool {
	if objectID == document.rootID {
		return true
	}
	_, isPageTree := document.pageTreeIDs[objectID]
	return isPageTree
}

func sortedObjectIDs(objects map[int]pdfObject) []int {
	ids := make([]int, 0, len(objects))
	for id := range objects {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func rewriteObjectReferences(content []byte, source int, objectMap map[objectKey]int) []byte {
	return rewriteOutsideStreams(content, func(segment []byte) []byte {
		return indirectRefRe.ReplaceAllFunc(segment, func(match []byte) []byte {
			parts := indirectRefRe.FindSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			oldObjectID, err := strconv.Atoi(string(parts[1]))
			if err != nil {
				return match
			}
			newObjectID, ok := objectMap[objectKey{source: source, number: oldObjectID}]
			if !ok {
				return match
			}
			return fmt.Appendf(nil, "%d 0 R", newObjectID)
		})
	})
}

func replacePageParent(content []byte, parentObjectID int) []byte {
	return rewriteOutsideStreams(content, func(segment []byte) []byte {
		replacement := fmt.Appendf(nil, "/Parent %d 0 R", parentObjectID)
		return parentRefRe.ReplaceAll(segment, replacement)
	})
}

func rewriteOutsideStreams(content []byte, rewrite func([]byte) []byte) []byte {
	streamStart := bytes.Index(content, []byte("stream"))
	if streamStart < 0 {
		return rewrite(content)
	}
	streamEnd := bytes.LastIndex(content, []byte("endstream"))
	if streamEnd < streamStart {
		return rewrite(content)
	}
	streamEnd += len("endstream")

	var out bytes.Buffer
	out.Write(rewrite(content[:streamStart]))
	out.Write(content[streamStart:streamEnd])
	out.Write(rewrite(content[streamEnd:]))
	return out.Bytes()
}

func renderMergedPDF(version string, pageIDs []int, copiedObjects map[int][]byte, nextObjectID int) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "%%PDF-%s\n", version)

	offsets := make([]int, nextObjectID)
	writeObject := func(objectID int, content []byte) {
		offsets[objectID] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", objectID)
		out.Write(bytes.TrimSpace(content))
		out.WriteString("\nendobj\n")
	}

	writeObject(mergedCatalogObjectID, []byte("<<\n/Type /Catalog\n/Pages 2 0 R\n>>"))
	writeObject(mergedPagesObjectID, []byte(pagesTreeContent(pageIDs)))
	for objectID := firstCopiedObjectID; objectID < nextObjectID; objectID++ {
		writeObject(objectID, copiedObjects[objectID])
	}

	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", nextObjectID)
	out.WriteString("0000000000 65535 f \n")
	for objectID := 1; objectID < nextObjectID; objectID++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[objectID])
	}
	fmt.Fprintf(&out, "trailer\n<<\n/Size %d\n/Root 1 0 R\n>>\nstartxref\n%d\n%%%%EOF\n", nextObjectID, xrefOffset)
	return out.Bytes()
}

func pagesTreeContent(pageIDs []int) string {
	var kids bytes.Buffer
	for _, pageID := range pageIDs {
		fmt.Fprintf(&kids, "%d 0 R ", pageID)
	}
	return fmt.Sprintf("<<\n/Type /Pages\n/Kids [%s]\n/Count %d\n>>", kids.String(), len(pageIDs))
}

func maxPDFVersion(documents []*pdfDocument) string {
	maxMajor := 1
	maxMinor := 3
	for _, document := range documents {
		major, minor := parseVersionParts(document.version)
		if major > maxMajor || (major == maxMajor && minor > maxMinor) {
			maxMajor = major
			maxMinor = minor
		}
	}
	return fmt.Sprintf("%d.%d", maxMajor, maxMinor)
}

func parseVersionParts(version string) (int, int) {
	var major, minor int
	_, err := fmt.Sscanf(version, "%d.%d", &major, &minor)
	if err != nil {
		return 1, 3
	}
	return major, minor
}
