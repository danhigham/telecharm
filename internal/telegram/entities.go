package telegram

import (
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/gotd/td/tg"
)

// entityAnnotation represents a markdown wrapper to apply at a UTF-16 offset range.
type entityAnnotation struct {
	offset int // UTF-16 code unit offset
	length int // UTF-16 code unit length
	prefix string
	suffix string
}

// EntitiesToMarkdown converts a Telegram message's plain text and entity list
// into a markdown-formatted string suitable for glamour rendering.
//
// Telegram entities use UTF-16 code unit offsets, so we convert to UTF-16,
// apply annotations, then convert back to UTF-8.
func EntitiesToMarkdown(text string, entities []tg.MessageEntityClass) string {
	if len(entities) == 0 {
		return text
	}

	// Build annotations from entities.
	annotations := make([]entityAnnotation, 0, len(entities))
	for _, e := range entities {
		a, ok := entityToAnnotation(text, e)
		if !ok {
			continue
		}
		annotations = append(annotations, a)
	}

	if len(annotations) == 0 {
		return text
	}

	// Sort by offset, then by length descending (longer entities first for nesting).
	sort.Slice(annotations, func(i, j int) bool {
		if annotations[i].offset != annotations[j].offset {
			return annotations[i].offset < annotations[j].offset
		}
		return annotations[i].length > annotations[j].length
	})

	// Convert text to UTF-16 code units for correct offset handling.
	runes := []rune(text)
	utf16Units := utf16.Encode(runes)

	// Build result by walking through UTF-16 units and inserting annotations.
	// We collect open/close markers at each position.
	type marker struct {
		pos    int    // UTF-16 position
		text   string // prefix or suffix to insert
		isOpen bool   // true for prefix (opening), false for suffix (closing)
		index  int    // original annotation index for stable sort
	}

	var markers []marker
	for i, a := range annotations {
		markers = append(markers, marker{pos: a.offset, text: a.prefix, isOpen: true, index: i})
		end := a.offset + a.length
		if end > len(utf16Units) {
			end = len(utf16Units)
		}
		markers = append(markers, marker{pos: end, text: a.suffix, isOpen: false, index: i})
	}

	// Sort markers: by position, then closing before opening at same position,
	// and for closings at same position, reverse order (last opened closes first).
	sort.SliceStable(markers, func(i, j int) bool {
		if markers[i].pos != markers[j].pos {
			return markers[i].pos < markers[j].pos
		}
		// At same position: closings before openings.
		if markers[i].isOpen != markers[j].isOpen {
			return !markers[i].isOpen
		}
		// Both closings at same position: reverse annotation order.
		if !markers[i].isOpen {
			return markers[i].index > markers[j].index
		}
		// Both openings at same position: keep annotation order.
		return markers[i].index < markers[j].index
	})

	// Build the final string.
	var b strings.Builder
	markerIdx := 0
	for i := 0; i <= len(utf16Units); i++ {
		// Insert any markers at this position.
		for markerIdx < len(markers) && markers[markerIdx].pos == i {
			b.WriteString(markers[markerIdx].text)
			markerIdx++
		}
		// Append the character at this position.
		if i < len(utf16Units) {
			// Decode UTF-16 unit(s) back to rune.
			if utf16.IsSurrogate(rune(utf16Units[i])) {
				if i+1 < len(utf16Units) {
					r := utf16.DecodeRune(rune(utf16Units[i]), rune(utf16Units[i+1]))
					b.WriteRune(r)
					i++ // skip the low surrogate
				}
			} else {
				b.WriteRune(rune(utf16Units[i]))
			}
		}
	}

	return b.String()
}

// entityToAnnotation converts a Telegram entity into a prefix/suffix annotation.
func entityToAnnotation(text string, entity tg.MessageEntityClass) (entityAnnotation, bool) {
	offset := entity.GetOffset()
	length := entity.GetLength()

	switch e := entity.(type) {
	case *tg.MessageEntityBold:
		return entityAnnotation{offset, length, "**", "**"}, true
	case *tg.MessageEntityItalic:
		return entityAnnotation{offset, length, "*", "*"}, true
	case *tg.MessageEntityCode:
		return entityAnnotation{offset, length, "`", "`"}, true
	case *tg.MessageEntityPre:
		lang := e.Language
		return entityAnnotation{offset, length, "```" + lang + "\n", "\n```"}, true
	case *tg.MessageEntityStrike:
		return entityAnnotation{offset, length, "~~", "~~"}, true
	case *tg.MessageEntityUnderline:
		// Markdown doesn't have underline; use emphasis as fallback.
		return entityAnnotation{offset, length, "*", "*"}, true
	case *tg.MessageEntityTextURL:
		return entityAnnotation{offset, length, "[", "](" + e.URL + ")"}, true
	case *tg.MessageEntityURL:
		// Auto-linked URL — extract the URL text and make it a proper link.
		urlText := extractUTF16Substring(text, offset, length)
		return entityAnnotation{offset, length, "[", "](" + urlText + ")"}, true
	case *tg.MessageEntityMentionName:
		return entityAnnotation{offset, length, "**", "**"}, true
	case *tg.MessageEntityBlockquote:
		return entityAnnotation{offset, length, "> ", ""}, true
	case *tg.MessageEntitySpoiler:
		// No markdown equivalent; render with indicator.
		return entityAnnotation{offset, length, "||", "||"}, true
	case *tg.MessageEntityMention:
		// @username — already visible in text, make bold.
		return entityAnnotation{offset, length, "**", "**"}, true
	case *tg.MessageEntityHashtag:
		return entityAnnotation{offset, length, "**", "**"}, true
	case *tg.MessageEntityBotCommand:
		return entityAnnotation{offset, length, "`", "`"}, true
	case *tg.MessageEntityEmail:
		emailText := extractUTF16Substring(text, offset, length)
		return entityAnnotation{offset, length, "[", "](mailto:" + emailText + ")"}, true
	default:
		// Unknown entity types are passed through unchanged.
		return entityAnnotation{}, false
	}
}

// extractUTF16Substring extracts a substring from text using UTF-16 offset and length.
func extractUTF16Substring(text string, offset, length int) string {
	runes := []rune(text)
	utf16Units := utf16.Encode(runes)

	end := offset + length
	if end > len(utf16Units) {
		end = len(utf16Units)
	}
	if offset >= len(utf16Units) {
		return ""
	}

	subUnits := utf16Units[offset:end]
	subRunes := utf16.Decode(subUnits)
	return string(subRunes)
}
