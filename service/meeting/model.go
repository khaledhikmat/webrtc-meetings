package meeting

type Meeting struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Duration     int64  `json:"duration"`
	Participants int64  `json:"participants"`
	Active       bool   `json:"isActive"`
}
