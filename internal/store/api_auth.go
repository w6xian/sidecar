package store

type AuthInfoLite struct {
	Id        int64  `json:"id"`
	ApiKey    string `json:"api_key"`
	ApiSecret string `json:"api_secret"`
	AppId     string `json:"app_id"`
	MchId     string `json:"mch_id"`
	Status    int64  `json:"status"`
	Tracker   *Tracker
}

func (a *AuthInfoLite) Mask() {
	a.ApiSecret = "********"
}
func (a *AuthInfoLite) GetHeader() map[string]string {
	return map[string]string{
		"app_id":     a.AppId,
		"mch_id":     a.MchId,
		"api_key":    a.ApiKey,
		"api_secret": a.ApiSecret,
	}
}

type UpdateApiAuthReq struct {
	Id      int64    `json:"id"`
	Key     string   `json:"key"`
	Value   string   `json:"value"`
	Tracker *Tracker `json:"-"`
}

func (req *UpdateApiAuthReq) Validate() error {
	return nil
}
