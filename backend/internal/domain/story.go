package domain

import "time"

type StylePrefix string

type EndingType string

const (
	EndingVictory EndingType = "victory"
	EndingDefeat  EndingType = "defeat"
	EndingTwist   EndingType = "twist"
)

type Choice struct {
	Label string `json:"label"`
}

type Page struct {
	Index          int        `json:"index"`
	Narrative      string     `json:"narrative"`
	ImagePrompt    string     `json:"-"`
	ImageURL       string     `json:"imageUrl,omitempty"`
	ImageProvider  string     `json:"imageProvider,omitempty"`
	AudioURL       string     `json:"audioUrl,omitempty"`
	DepthURL       string     `json:"depthUrl,omitempty"`
	Choices        []Choice   `json:"choices"`
	IsEnding       bool       `json:"isEnding"`
	EndingType     EndingType `json:"endingType,omitempty"`
	RunningSummary string     `json:"-"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type Story struct {
	ID          string      `json:"storyId"`
	UserID      string      `json:"userId,omitempty"`
	Topic       string      `json:"topic"`
	StylePrefix StylePrefix `json:"stylePrefix"`
	Pages       []Page      `json:"pages"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

func (s *Story) Current() *Page {
	if len(s.Pages) == 0 {
		return nil
	}
	return &s.Pages[len(s.Pages)-1]
}

// PageDraft is the structured output from a StoryProvider -- no image yet.
type PageDraft struct {
	Narrative      string
	ImagePrompt    string
	Choices        []Choice
	IsEnding       bool
	EndingType     EndingType
	RunningSummary string
}
