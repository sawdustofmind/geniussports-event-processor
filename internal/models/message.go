package models

import "time"

type Message struct {
	Header                     Header                      `json:"Header"`
	AmericanFootballMatchState *AmericanFootballMatchState `json:"AmericanFootballMatchState,omitempty"`
	Fixture                    *Fixture                    `json:"Fixture,omitempty"`
}

type Header struct {
	Retry        int       `json:"Retry"`
	MessageGuid  string    `json:"MessageGuid"`
	TimeStampUtc time.Time `json:"TimeStampUtc"`
}

type AmericanFootballMatchState struct {
	Score  Score  `json:"Score"`
	Period Period `json:"Period"`

	GameTime struct {
		Clock          string    `json:"Clock"`
		IsRunning      bool      `json:"IsRunning"`
		LastUpdatedUtc time.Time `json:"LastUpdatedUtc"`
	} `json:"GameTime"`

	FixtureId string `json:"FixtureId"`
	// TODO: do we suppose to skip this update entirely?
	// IsReliable bool   `json:"IsReliable"`
}

type Score struct {
	Away        int  `json:"Away"`
	Home        int  `json:"Home"`
	IsConfirmed bool `json:"IsConfirmed"`
}

type Period struct {
	Type   string `json:"Type"`
	Number int    `json:"Number"`
}
type Fixture struct {
	ID           int          `json:"Id"`
	Competitors  []Competitor `json:"Competitors"`
	Status       string       `json:"Status"`
	StartTimeUtc time.Time    `json:"StartTimeUtc"`
}

type Competitor struct {
	ID       int    `json:"Id"`
	Name     string `json:"Name"`
	HomeAway string `json:"HomeAway"`
}
