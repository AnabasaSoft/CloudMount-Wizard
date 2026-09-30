package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// latestReleaseURL devuelve la última release publicada (excluye borradores y pre-releases)
const latestReleaseURL = "https://api.github.com/repos/AnabasaSoft/CloudMount-Wizard/releases/latest"

// Release es la información de una release de GitHub que nos interesa
type Release struct {
	Tag string `json:"tag_name"` // Ej: "v1.2.3"
	URL string `json:"html_url"` // Página de la release
}

// CheckLatest consulta GitHub y devuelve la última release si es más nueva que
// currentVersion. Si no hay versión nueva devuelve nil sin error.
func CheckLatest(currentVersion string) (*Release, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", latestReleaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "CloudMount-Wizard/"+currentVersion) // GitHub exige User-Agent

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub devolvió %s", resp.Status)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if IsNewer(rel.Tag, currentVersion) {
		return &rel, nil
	}
	return nil, nil
}

// IsNewer indica si la versión latest es mayor que current (formato "v1.2.3" o "1.2.3").
// Si alguna no se puede interpretar, devuelve false para no avisar por error.
func IsNewer(latest, current string) bool {
	l, okL := parseVersion(latest)
	c, okC := parseVersion(current)
	if !okL || !okC {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// parseVersion convierte "v1.2.3" en [1 2 3]; las partes que falten cuentan como 0
func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, _, _ = strings.Cut(v, "-") // Ignoramos sufijos tipo "-beta"
	parts := strings.Split(v, ".")
	if v == "" || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
