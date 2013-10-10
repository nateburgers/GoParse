package parse

import (
	"unicode"
)

type Result interface{}
type ParseResult struct {
	Result    interface{}
	Remainder []byte
}
type Parser func([]byte) []ParseResult
type ParserGenerator func(interface{}) Parser
type Predicate func(interface{}) bool

func (self ParseResult) Stringify() string {
	return string(self.Result.([]byte))
}

// helpers
func fold(f func(Parser, Parser) Parser, xs []Parser) Parser {
	memo := xs[0]
	for _, x := range xs[1:] {
		memo = f(memo, x)
	}
	return memo
}

func cons(a interface{}, b interface{}) []interface{} {
	switch b := b.(type) {
	case []interface{}:
		return append([]interface{}{a}, b...)
	default:
		return append([]interface{}{a}, b)
	}
}

func Unit(object Result) Parser {
	return func(input []byte) []ParseResult {
		return []ParseResult{ParseResult{object, input}}
	}
}

func Fail(input []byte) []ParseResult {
	return make([]ParseResult, 0)
}

func Bind(f Parser, g ParserGenerator) Parser {
	return func(input []byte) []ParseResult {
		results := make([]ParseResult, 0)
		resultsOfF := f(input)
		for _, resultOfF := range resultsOfF {
			results = append(results, g(resultOfF.Result)(resultOfF.Remainder)...)
		}
		return results
	}
}

func Using(f Parser, g func(interface{}) interface{}) Parser {
	return Bind(f, func(resultOfF interface{}) Parser {
		return Unit(g(resultOfF))
	})
}

func Pred(p Predicate) Parser {
	return func(input []byte) []ParseResult {
		if len(input) == 0 {
			return Fail(input)
		}
		if p(input[0]) {
			return Unit(input[0])(input[1:])
		} else {
			return Fail(input)
		}
	}
}

func Literal(object interface{}) Parser {
	return Pred(func(testObject interface{}) bool {
		return testObject == object
	})
}

func or(f Parser, g Parser) Parser {
	return func(input []byte) []ParseResult {
		return append(f(input), g(input)...)
	}
}

func Or(parsers ...Parser) Parser {
	return fold(or, parsers)
}

func xor(f Parser, g Parser) Parser {
	return func(input []byte) []ParseResult {
		resultsOfF := f(input)
		if len(resultsOfF) > 0 {
			return resultsOfF
		} else {
			return g(input)
		}
	}
}

func Xor(parsers ...Parser) Parser {
	return fold(xor, parsers)
}

func and(f Parser, g Parser) Parser {
	return Bind(f, func(resultOfF interface{}) Parser {
		return Bind(g, func(resultOfG interface{}) Parser {
			return Unit([]interface{}{resultOfF, resultOfG})
		})
	})
}

func And(parsers ...Parser) Parser {
	return fold(and, parsers)
}

func Many(f Parser) Parser {
	return func(input []byte) []ParseResult {
		return Or(f, Using(And(f, Many(f)), func(x interface{}) interface{} {
			both := x.([]interface{})
			return cons(both[0], both[1])
		}))(input)
	}
}

func XMany(f Parser) Parser {
	return func(input []byte) []ParseResult {
		return Xor(Using(And(f, XMany(f)), func(x interface{}) interface{} {
			both := x.([]interface{})
			return cons(both[0], both[1])
		}), f)(input)
	}
}

func ThenIgnore(f Parser, g Parser) Parser {
	return Bind(f, func(resultOfF interface{}) Parser {
		return Bind(g, func(resultOfG interface{}) Parser {
			return Unit(resultOfF)
		})
	})
}

func IgnoreThen(f Parser, g Parser) Parser {
	return Bind(f, func(resultOfF interface{}) Parser {
		return Bind(g, func(resultOfG interface{}) Parser {
			return Unit(resultOfG)
		})
	})
}

func ThenX(f Parser, g Parser) Parser {
	return Or(f, ThenIgnore(f, g))
}

func XThen(f Parser, g Parser) Parser {
	return Or(g, IgnoreThen(f, g))
}

func XThenX(f Parser, g Parser, h Parser) Parser {
	return XThen(f, ThenX(g, h))
}

func SeparateBy(f Parser, g Parser) Parser {
	return And(f, XMany(IgnoreThen(g, f)))
}

// Actual Parsers
func wrapUnicodePredicate(p func(rune) bool) Parser {
	return Pred(func(object interface{}) bool {
		return p(rune(object.(byte)))
	})
}
func Letter(input []byte) []ParseResult {
	return wrapUnicodePredicate(unicode.IsLetter)(input)
}

func Digit(input []byte) []ParseResult {
	return wrapUnicodePredicate(unicode.IsDigit)(input)
}

func Whitespace(input []byte) []ParseResult {
	return Many(wrapUnicodePredicate(unicode.IsSpace))(input)
}

func Word(input []byte) []ParseResult {
	// return Many(Letter)(input)
	return XMany(Letter)(input)
}

func Integer(input []byte) []ParseResult {
	return Many(Digit)(input)
}
