package gopenqa

/* Product */
type Product struct {
	Arch     string            `json:"arch"`
	Distri   string            `json:"distri"`
	Flavor   string            `json:"flavor"`
	Group    string            `json:"group"`
	ID       int               `json:"id"`
	Version  string            `json:"version"`
	Settings map[string]string `json:"settings"`
}
