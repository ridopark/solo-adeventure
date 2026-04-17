package domain

import (
	"errors"
	"strings"
)

// ErrUnsafeTopic is returned when a submitted topic fails the server-side
// safety filter before any LLM call. The HTTP layer maps this to 400 with a
// generic "try a different topic" message -- do NOT surface which rule tripped,
// to avoid giving attackers a feedback signal for bypass.
var ErrUnsafeTopic = errors.New("topic rejected by safety filter")

// ErrTopicLength is returned when the topic is too short or too long.
var ErrTopicLength = errors.New("topic length out of bounds")

// ErrUnsafeImagePrompt is returned when the LLM-generated imagePrompt fails
// the pre-dispatch filter. The caller should skip image generation and log a
// warning; the page still renders with no image.
var ErrUnsafeImagePrompt = errors.New("image prompt rejected by safety filter")

const (
	MinTopicLen = 3
	MaxTopicLen = 200
)

// topicBlocklist is intentionally narrow: only patterns that are
// near-unambiguous markers of disallowed content. The system prompt handles
// borderline topics via silent reinterpretation. Keep false-positive rate low.
var topicBlocklist = []string{
	// CSAM and sexualized minors -- zero tolerance
	"child porn", "childporn", "cp child",
	"underage sex", "underage porn",
	"kiddie porn", "kiddy porn",
	"pedo", "pedophile", "paedophile",
	"csam",
	"lolicon", "shotacon",
	"loli sex", "shota sex",
	"minor sex", "sex with minor",
	"child rape", "rape child",
	"molest child", "groom child", "grooming minor",
	"schoolgirl sex", "teen rape",

	// Explicit adult fiction intent
	"xxx story", "porn story", "hentai",
	"erotic roleplay", "nsfw roleplay",

	// Real-world harm instructions (topic-level; narrative-level is covered by system prompt)
	"how to make a bomb", "how to synthesize",
	"build a bomb", "assemble a bomb",
	"school shooting", "mass shooting plan",
	"suicide method", "kill myself",

	// Real-person sexual content is hard to detect; skip the blocklist and rely on Claude's refusal.
}

// imagePromptBlocklist is a defense-in-depth filter on the LLM-generated
// imagePrompt before it reaches Together or fal. Claude's system prompt already
// forbids this material, but we add a server-side tripwire for prompt-injection
// and jailbreak attempts. Kept narrow to minimize false positives; fal's own
// enable_safety_checker handles post-generation output.
var imagePromptBlocklist = []string{
	"naked body", "nude body", "fully nude", "fully naked",
	"bare breasts", "exposed breasts", "topless woman", "topless girl", "topless boy",
	"penis", "vagina", "vulva", "erection", "erect phallus",
	"naked child", "nude child", "child naked", "child nude",
	"nude minor", "naked minor", "minor nude", "minor naked",
	"young girl naked", "young boy naked", "young girl nude", "young boy nude",
	"loli", "shota",
	"decapitated", "disemboweled", "severed head", "flayed corpse", "eviscerated",
	"bomb diagram", "weapon schematic", "explosive blueprint",
}

// ValidateImagePrompt returns ErrUnsafeImagePrompt if the prompt contains any
// blocklisted phrase. Case-insensitive substring match.
func ValidateImagePrompt(prompt string) error {
	lower := strings.ToLower(prompt)
	for _, bad := range imagePromptBlocklist {
		if strings.Contains(lower, bad) {
			return ErrUnsafeImagePrompt
		}
	}
	return nil
}

// ValidateTopic returns nil if the topic is acceptable, or an error otherwise.
// The error is either ErrTopicLength or ErrUnsafeTopic -- never a detailed reason.
func ValidateTopic(topic string) error {
	t := strings.TrimSpace(topic)
	if len(t) < MinTopicLen || len(t) > MaxTopicLen {
		return ErrTopicLength
	}
	lower := strings.ToLower(t)
	for _, bad := range topicBlocklist {
		if strings.Contains(lower, bad) {
			return ErrUnsafeTopic
		}
	}
	return nil
}
