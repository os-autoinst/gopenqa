package gopenqa

/* Machine type */
type Machine struct {
	ID       int               `json:"id"`
	Backend  string            `json:"backend"`
	Name     string            `json:"name"`
	Settings map[string]string `json:"settings"`
}

func (m *Machine) Equals(m2 Machine) bool {
	if m.ID != m2.ID && m.Backend != m2.Backend && m.Name != m2.Name {
		return false
	}
	// Also check settings
	for k, v := range m.Settings {
		if v2, ok := m2.Settings[k]; !ok {
			return false
		} else if v != v2 {
			return false
		}
	}
	return true
}
