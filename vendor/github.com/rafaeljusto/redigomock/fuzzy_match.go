package redigomock

import "reflect"

// FuzzyMatcher is an interface that exports exports one function. It can be
// passed to the Command as a argument. When the command is evaluated agains
// data provided in mock connection Do call, FuzzyMatcher will call Match on the
// argument and returns true if argument fulfils constraints set in concrete
// implementation
type FuzzyMatcher interface {

	// Match takes an argument passed to mock connection Do call and check if
	// it fulfills constraints set in concrete implementation of this interface
	Match(interface{}) bool
}

// NewAnyInt returns a FuzzyMatcher instance matching any integer passed as an
// argument
func NewAnyInt() FuzzyMatcher {
	return anyInt{}
}

// NewAnyDouble returns a FuzzyMatcher instance mathing any double passed as
// an argument
func NewAnyDouble() FuzzyMatcher {
	return anyDouble{}
}

// NewAnyData returns a FuzzyMatcher instance matching every data passed as
// an arguments (returns true by default)
func NewAnyData() FuzzyMatcher {
	return anyData{}
}

type anyInt struct{}

func (matcher anyInt) Match(input interface{}) bool {
	switch input.(type) {
	case int, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

type anyDouble struct{}

func (matcher anyDouble) Match(input interface{}) bool {
	switch input.(type) {
	case float32, float64:
		return true
	default:
		return false
	}
}

type anyData struct{}

func (matcher anyData) Match(input interface{}) bool {
	return true
}

func implementsFuzzy(input interface{}) bool {
	return reflect.TypeOf(input).Implements(reflect.TypeOf((*FuzzyMatcher)(nil)).Elem())
}
