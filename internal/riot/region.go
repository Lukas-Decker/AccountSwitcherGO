package riot

import (
	"errors"
	"sort"
	"strings"
)

// ErrUnknownRegion is returned for a region code the package does not know.
var ErrUnknownRegion = errors.New("riot: unknown region")

// Route is one of the four regional clusters account-v1 is served from.
type Route string

const (
	RouteAmericas Route = "americas"
	RouteAsia     Route = "asia"
	RouteEurope   Route = "europe"
	RouteSEA      Route = "sea"
)

// Region ties together the three different names the same place goes by.
//
// Riot splits its endpoints two ways and the split is not cosmetic. account-v1
// answers on a regional cluster (europe), while summoner-v4 and league-v4 answer
// on a platform host (euw1). Sending either to the other's host returns 404 with
// no hint as to why, so both belong on the region rather than being derived at
// the call site. The community sites then use a third set of slugs again.
type Region struct {
	// Platform is the shard id used by summoner-v4 and league-v4, e.g. "euw1".
	Platform string
	// Route is the cluster used by account-v1, e.g. "europe".
	Route Route
	// OPGG is the slug op.gg, u.gg and friends use in their paths, e.g. "euw".
	OPGG string
	// Display is the human-facing label.
	Display string
}

// Host returns the platform host summoner-v4 and league-v4 are served from.
func (r Region) Host() string { return r.Platform + ".api.riotgames.com" }

// RouteHost returns the regional host account-v1 is served from.
func (r Region) RouteHost() string { return string(r.Route) + ".api.riotgames.com" }

// regions is keyed by platform id.
var regions = map[string]Region{
	"br1":  {Platform: "br1", Route: RouteAmericas, OPGG: "br", Display: "Brazil"},
	"eun1": {Platform: "eun1", Route: RouteEurope, OPGG: "eune", Display: "EU Nordic & East"},
	"euw1": {Platform: "euw1", Route: RouteEurope, OPGG: "euw", Display: "EU West"},
	"jp1":  {Platform: "jp1", Route: RouteAsia, OPGG: "jp", Display: "Japan"},
	"kr":   {Platform: "kr", Route: RouteAsia, OPGG: "kr", Display: "Korea"},
	"la1":  {Platform: "la1", Route: RouteAmericas, OPGG: "lan", Display: "Latin America North"},
	"la2":  {Platform: "la2", Route: RouteAmericas, OPGG: "las", Display: "Latin America South"},
	"me1":  {Platform: "me1", Route: RouteEurope, OPGG: "me", Display: "Middle East"},
	"na1":  {Platform: "na1", Route: RouteAmericas, OPGG: "na", Display: "North America"},
	"oc1":  {Platform: "oc1", Route: RouteSEA, OPGG: "oce", Display: "Oceania"},
	"ru":   {Platform: "ru", Route: RouteEurope, OPGG: "ru", Display: "Russia"},
	"sg2":  {Platform: "sg2", Route: RouteSEA, OPGG: "sg", Display: "Singapore & Malaysia"},
	"tr1":  {Platform: "tr1", Route: RouteEurope, OPGG: "tr", Display: "Turkey"},
	"tw2":  {Platform: "tw2", Route: RouteSEA, OPGG: "tw", Display: "Taiwan"},
	"vn2":  {Platform: "vn2", Route: RouteSEA, OPGG: "vn", Display: "Vietnam"},
}

// aliases maps the other spellings people actually type, including every op.gg
// slug, so a region pasted from a profile URL resolves.
var aliases = map[string]string{
	"eune": "eun1", "euw": "euw1", "jp": "jp1", "lan": "la1", "las": "la2",
	"na": "na1", "oce": "oc1", "oc": "oc1", "br": "br1", "tr": "tr1",
	"sg": "sg2", "tw": "tw2", "vn": "vn2", "me": "me1",
}

// LookupRegion resolves a platform id, an op.gg slug or a common alias.
func LookupRegion(code string) (Region, error) {
	key := strings.ToLower(strings.TrimSpace(code))
	if key == "" {
		return Region{}, ErrUnknownRegion
	}
	if r, ok := regions[key]; ok {
		return r, nil
	}
	if platform, ok := aliases[key]; ok {
		return regions[platform], nil
	}
	return Region{}, ErrUnknownRegion
}

// Regions lists every known region, ordered for display.
func Regions() []Region {
	out := make([]Region, 0, len(regions))
	for _, r := range regions {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Display < out[j].Display })
	return out
}
