package triposter

type Configuration struct {
	Conf    conf              `json:"conf"`
	Objects map[string]object `json:"objects"`
}
type object struct {
	Ref    string `json:"ref"`
	Type   string `json:"type"`
	Source int    `json:"source"`
}
type conf struct {
	Host    string `json:"host"`
	Period  string `json:"period"`
	Site_id int    `json:"site_id"`
}
