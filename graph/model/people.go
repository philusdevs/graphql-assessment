package model

type People struct {
	Name      string `json:"name"`
	Height    string `json:"height"`
	Mass      string `json:"mass"`
	Gender    string `json:"gender"`
	Homeworld string `json:"homeworld"`
}

type Results struct {
	Peoples []*People `json:"results"`
}
