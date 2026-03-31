package store

type Header struct {
	TrackId string `json:"track_id"`
	AppId   string `json:"app_id"`
	Lang    string `json:"lang"`
	Auth    string `json:"auth"`
}

func NewHeader(tracker *Tracker) *Header {
	return &Header{
		TrackId: tracker.TrackId,
		AppId:   tracker.AppId,
		Lang:    tracker.Language,
		Auth:    tracker.JwtToken,
	}
}
