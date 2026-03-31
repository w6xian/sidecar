package options

type App struct {
	Origins []string `json:"origins"`
	Mode    string   `json:"mode"`
}

type Apps struct {
	App     App    `json:"app"`
	Version string `json:"version"`
}

func (app *Apps) GetVserion() string {
	return ""
}
