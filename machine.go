package gopenqa

/* Machine type */
type Machine struct {
	ID      int    `json: "id"`
	Backend string `json: "backend"`
	Name    string `json: "name"`
}
