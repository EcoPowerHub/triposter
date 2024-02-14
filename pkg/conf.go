package triposter

type Configuration struct {
	Conf    conf              `json:"conf"`
	Objects map[string]object `json:"objects"`
}
type object struct {
	Ref  string `json:"ref"`
	Type string `json:"type"`
}
type conf struct {
	Host        string `json:"host"`
	PostPeriodS int    `json:"post_period_s"`
}
