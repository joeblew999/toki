package markdown

import (
	"fmt"

	"github.com/cespare/xxhash/v2"
	"github.com/romshark/toki/internal/arb"
)

// HashText generates a stable hash-based ID for a markdown text.
// Uses the same format as codeparse: "msg" + hex(xxhash64(content))
func HashText(hasher *xxhash.Digest, content string) string {
	hasher.Reset()
	_, _ = hasher.WriteString(content)
	return fmt.Sprintf("msg%x", hasher.Sum64())
}

// ToARBMessage converts a markdown Text to an ARB Message.
func (t *Text) ToARBMessage(hasher *xxhash.Digest) arb.Message {
	id := HashText(hasher, t.Content)

	msg := arb.Message{
		ID:         id,
		ICUMessage: t.Content, // Plain text, no ICU formatting needed
	}

	// Add description based on type and context
	desc := string(t.Type)
	if t.Context != "" {
		desc = t.Context
	}
	msg.Description = desc

	return msg
}

// ToARBMessages converts all texts from a file to ARB messages.
func (f *File) ToARBMessages(hasher *xxhash.Digest) []arb.Message {
	messages := make([]arb.Message, 0, len(f.Texts))
	for i := range f.Texts {
		msg := f.Texts[i].ToARBMessage(hasher)
		// Add file context to the message
		msg.Context = f.Path
		messages = append(messages, msg)
	}
	return messages
}

// ToARBMessages converts all scan results to ARB messages.
func (r *ScanResult) ToARBMessages(hasher *xxhash.Digest) []arb.Message {
	total := r.TotalTexts()
	messages := make([]arb.Message, 0, total)
	for _, f := range r.Files {
		messages = append(messages, f.ToARBMessages(hasher)...)
	}
	return messages
}

// GenerateHashes populates the IDHash field on all texts in the result.
func (r *ScanResult) GenerateHashes(hasher *xxhash.Digest) {
	for _, f := range r.Files {
		for i := range f.Texts {
			f.Texts[i].IDHash = HashText(hasher, f.Texts[i].Content)
		}
	}
}
