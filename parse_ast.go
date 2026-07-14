package isnow

import (
	"strconv"

	g "github.com/uplang/isnow.go/internal/isnowgrammar"
)

// field builds a present field: an optional '!' exclusion over its terms.
func field(ctx g.IFieldContext) rawField {
	terms := ctx.AllTerm()
	out := rawField{present: true, exclude: ctx.BANG() != nil, terms: make([]rawTerm, len(terms))}
	for i, t := range terms {
		out.terms[i] = term(t)
	}
	return out
}

// term reads `DASH? atom (DASH atom)? incr?` or a lone incr.
func term(ctx g.ITermContext) rawTerm {
	atoms := ctx.AllAtom()
	if len(atoms) == 0 {
		return rawTerm{incr: incr(ctx.Incr())}
	}
	dashes := len(ctx.AllDASH())
	out := rawTerm{lo: atom(atoms[0]), loFromEnd: dashes > len(atoms)-1, incr: incrOrNil(ctx.Incr())}
	if len(atoms) == 2 {
		out.hi = atom(atoms[1])
	}
	return out
}

func incrOrNil(ctx g.IIncrContext) *rawIncr {
	if ctx == nil {
		return nil
	}
	return incr(ctx)
}

func incr(ctx g.IIncrContext) *rawIncr {
	return &rawIncr{fromEnd: ctx.DASH() != nil, qtys: qtys(ctx.AllQty())}
}

// atom reads a wildcard, a NAME symbol, or a numeric quantity run.
func atom(ctx g.IAtomContext) *rawAtom {
	switch {
	case ctx.STAR() != nil:
		return &rawAtom{star: true}
	case ctx.NAME() != nil:
		return &rawAtom{name: ctx.NAME().GetText()}
	default:
		return &rawAtom{qtys: qtys(ctx.AllQty())}
	}
}

func qtys(ctxs []g.IQtyContext) []rawQty {
	out := make([]rawQty, len(ctxs))
	for i, q := range ctxs {
		out[i] = qty(q)
	}
	return out
}

func qty(ctx g.IQtyContext) rawQty {
	// NUMBER is guaranteed by the grammar (qty : NUMBER NAME?); GetText is digits.
	text := ctx.NUMBER().GetText()
	n, err := strconv.Atoi(text)
	if err != nil {
		n = -1 // overflow/invalid: a negative magnitude fails every range check
	}
	return rawQty{num: n, digits: len(text), unit: qtyUnit(ctx)}
}

func qtyUnit(ctx g.IQtyContext) string {
	if ctx.NAME() == nil {
		return ""
	}
	return ctx.NAME().GetText()
}
