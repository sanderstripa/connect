package installprefs

// Preferences carries only the two presentation options selected during installation.
type Preferences struct {
	ID       string `json:"id"`
	Language string `json:"language"`
	Theme    string `json:"theme"`
}

const File = "installer-preferences.json"

func (p Preferences) Valid() bool {
	return p.ID != "" && (p.Language == "ru" || p.Language == "en") && (p.Theme == "light" || p.Theme == "dark")
}
