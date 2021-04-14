package gopenqa

/* Worker instance */
type Worker struct {
	Alive      int               `json:"alive"`
	Connected  int               `json:"connected"`
	Error      string            `json:"error"` // Error string if present
	Host       string            `json:"host"`
	ID         int               `json:"id"`
	Instance   int               `json:"instance"`
	Status     string            `json:"status"`
	Websocket  int               `json:"websocket"`
	Properties map[string]string `json:"properties"` // Worker properties as returned by openQA
}
