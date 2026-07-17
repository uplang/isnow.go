package domain

import isnow "github.com/tsvsheet/go-isnow"

// Canon returns the canonical form of src.
func Canon(src string) (string, error) {
	p, err := isnow.Parse(isnow.PatternText(src))
	if err != nil {
		return "", err
	}
	return p.Canonical(), nil
}

// Describe returns the canonical form and English explanation of src.
func Describe(src string) (Verdict, error) {
	p, err := isnow.Parse(isnow.PatternText(src))
	if err != nil {
		return Verdict{}, err
	}
	return Verdict{Canonical: p.Canonical(), Explain: p.Explain()}, nil
}
