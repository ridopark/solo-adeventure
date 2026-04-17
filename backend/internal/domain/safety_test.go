package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTopic(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		want  error
	}{
		{"empty", "", ErrTopicLength},
		{"too short", "ab", ErrTopicLength},
		{"too long", strings.Repeat("x", MaxTopicLen+1), ErrTopicLength},

		{"normal fantasy", "a lighthouse keeper in 1912", nil},
		{"dark but allowed", "a victorian ghost story with a vanished brother", nil},
		{"violent but within classic fiction", "a treasure hunt with pirates and a duel", nil},
		{"noir detective", "a detective tracking a murderer through prohibition-era chicago", nil},

		{"CSAM literal", "child porn tales", ErrUnsafeTopic},
		{"pedo marker", "stories of a pedophile grooming children", ErrUnsafeTopic},
		{"lolicon", "a lolicon story", ErrUnsafeTopic},
		{"sex with minor", "sex with minor girl at school", ErrUnsafeTopic},
		{"grooming minor", "grooming minor in a chat room", ErrUnsafeTopic},

		{"xxx story", "xxx story about two adults", ErrUnsafeTopic},
		{"nsfw roleplay", "nsfw roleplay with a teacher", ErrUnsafeTopic},

		{"bomb synth", "how to make a bomb at home", ErrUnsafeTopic},
		{"school shooting", "school shooting fiction", ErrUnsafeTopic},
		{"suicide method", "suicide method story", ErrUnsafeTopic},

		{"case insensitive", "CHILD PORN", ErrUnsafeTopic},
		{"embedded", "a fun lolicon lunchbreak", ErrUnsafeTopic},

		{"word child alone is fine", "a child who discovers a portal to another world", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTopic(tc.topic)
			if tc.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.True(t, errors.Is(err, tc.want), "got %v, want %v", err, tc.want)
		})
	}
}

func TestValidateImagePrompt(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   error
	}{
		{"benign", "a lighthouse at dusk, stormy sky, oil painting", nil},
		{"dark but allowed", "a gothic cathedral interior at midnight, candles flickering", nil},
		{"violent but classic", "pirates dueling on a ship deck, dramatic lighting", nil},

		{"nudity", "a naked body on a beach", ErrUnsafeImagePrompt},
		{"explicit", "erection in close-up", ErrUnsafeImagePrompt},
		{"minor unsafe", "naked child in a field", ErrUnsafeImagePrompt},
		{"loli", "a loli sitting in a room", ErrUnsafeImagePrompt},
		{"gore", "a severed head on a spike", ErrUnsafeImagePrompt},
		{"weapon schematic", "detailed bomb diagram", ErrUnsafeImagePrompt},

		{"bare does not match alone", "bare tree branches in winter", nil},
		{"child does not match alone", "a child watching fireflies", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateImagePrompt(tc.prompt)
			if tc.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.True(t, errors.Is(err, tc.want), "got %v, want %v", err, tc.want)
		})
	}
}
