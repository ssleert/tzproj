package conversions

import (
	"github.com/ssleert/tzproj/pkg/nationalize"
)

func CountryToIds(countries []nationalize.Country) []string {
	out := make([]string, 0, 32)
	for _, e := range countries {
		out = append(out, e.Id)
	}
	return out
}

func CountryToProbalities(countries []nationalize.Country) []float64 {
	out := make([]float64, 0, 32)
	for _, e := range countries {
		out = append(out, e.Probability)
	}
	return out
}
