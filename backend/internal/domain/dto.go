package domain

type StartStoryInput struct {
	Topic string `json:"topic"`
}

type StartStoryOutput struct {
	StoryID     string      `json:"storyId"`
	StylePrefix StylePrefix `json:"stylePrefix"`
	Page        Page        `json:"page"`
}

type ProgressInput struct {
	StoryID     string `json:"-"`
	ChoiceIndex int    `json:"choiceIndex"`
}

type ProgressOutput struct {
	Page Page `json:"page"`
}

type GenerateImageInput struct {
	Prompt      string      `json:"prompt"`
	StylePrefix StylePrefix `json:"stylePrefix"`
}

type GenerateImageOutput struct {
	URL      string `json:"url"`
	Provider string `json:"provider"`
}

type VisitInput struct {
	Path      string `json:"path"`
	Referrer  string `json:"referrer"`
	UserAgent string `json:"userAgent"`
}
