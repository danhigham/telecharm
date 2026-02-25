package telegram

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestEntitiesToMarkdown_NoEntities(t *testing.T) {
	text := "Hello world"
	result := EntitiesToMarkdown(text, nil)
	if result != text {
		t.Errorf("expected %q, got %q", text, result)
	}
}

func TestEntitiesToMarkdown_Bold(t *testing.T) {
	// "Hello world" with "world" bold (offset=6, length=5)
	text := "Hello world"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 6, Length: 5},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Hello **world**"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_Italic(t *testing.T) {
	text := "Hello world"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityItalic{Offset: 6, Length: 5},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Hello *world*"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_Code(t *testing.T) {
	text := "Use fmt.Println here"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityCode{Offset: 4, Length: 11},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Use `fmt.Println` here"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_Pre(t *testing.T) {
	text := "func main() {}"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityPre{Offset: 0, Length: 14, Language: "go"},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "```go\nfunc main() {}\n```"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_Strike(t *testing.T) {
	text := "Hello world"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityStrike{Offset: 6, Length: 5},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Hello ~~world~~"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_TextURL(t *testing.T) {
	text := "Click here for info"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityTextURL{Offset: 6, Length: 4, URL: "https://example.com"},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Click [here](https://example.com) for info"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_URL(t *testing.T) {
	text := "Visit https://example.com today"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityURL{Offset: 6, Length: 19},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Visit [https://example.com](https://example.com) today"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_BotCommand(t *testing.T) {
	text := "Type /start to begin"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBotCommand{Offset: 5, Length: 6},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Type `/start` to begin"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_MultipleEntities(t *testing.T) {
	// "Hello bold and italic world"
	text := "Hello bold and italic world"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 6, Length: 4},
		&tg.MessageEntityItalic{Offset: 15, Length: 6},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Hello **bold** and *italic* world"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_NestedBoldItalic(t *testing.T) {
	// "Hello world" with entire text bold and "world" also italic
	text := "Hello world"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 0, Length: 11},
		&tg.MessageEntityItalic{Offset: 6, Length: 5},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "**Hello *world***"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_Emoji(t *testing.T) {
	// Test with emoji (multi-byte UTF-16 characters).
	// "Hello 👋 world" — 👋 is U+1F44B, which is 2 UTF-16 code units (surrogate pair).
	text := "Hello 👋 world"
	// "world" starts at UTF-16 offset 9 (H=1,e=1,l=1,l=1,o=1, =1,👋=2, =1 => 9)
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 9, Length: 5},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Hello 👋 **world**"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_Blockquote(t *testing.T) {
	text := "This is quoted"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBlockquote{Offset: 0, Length: 14},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "> This is quoted"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_Email(t *testing.T) {
	text := "Email me at user@example.com"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityEmail{Offset: 12, Length: 16},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Email me at [user@example.com](mailto:user@example.com)"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEntitiesToMarkdown_Mention(t *testing.T) {
	text := "Hey @johndoe check this"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityMention{Offset: 4, Length: 8},
	}
	result := EntitiesToMarkdown(text, entities)
	expected := "Hey **@johndoe** check this"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractUTF16Substring(t *testing.T) {
	tests := []struct {
		text     string
		offset   int
		length   int
		expected string
	}{
		{"Hello world", 6, 5, "world"},
		{"Hello 👋 world", 9, 5, "world"},
		{"", 0, 0, ""},
		{"abc", 10, 5, ""}, // out of range
	}

	for _, tt := range tests {
		result := extractUTF16Substring(tt.text, tt.offset, tt.length)
		if result != tt.expected {
			t.Errorf("extractUTF16Substring(%q, %d, %d) = %q, want %q",
				tt.text, tt.offset, tt.length, result, tt.expected)
		}
	}
}
