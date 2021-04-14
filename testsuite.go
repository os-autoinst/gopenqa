package gopenqa

/* Test Suite */
type TestSuite struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Settings    map[string]string `json:"settings"`
}
