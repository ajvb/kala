package redigomock

import "testing"

func TestFuzzyCommandMatchAnyInt(t *testing.T) {
	var fuzzyCommandTestInput = []struct {
		arguments []interface{}
		match     bool
	}{
		{[]interface{}{"TEST_COMMAND", "Test string", 1}, true},
		{[]interface{}{"TEST_COMMAND", "Test string", 1234567}, true},
		{[]interface{}{"TEST_COMMAND", 1, "Test string"}, false},
		{[]interface{}{"TEST_COMMAND", "Test string", 1, 1}, false},
		{[]interface{}{"TEST_COMMAND", "AnotherString", 1}, false},
		{[]interface{}{"TEST_COMMAND", 1}, false},
		{[]interface{}{"TEST_COMMAND", "AnotherString"}, false},
		{[]interface{}{"TEST_COMMAND", "Test string", 1.0}, false},
		{[]interface{}{"TEST_COMMAND", "Test string", "This is not an int"}, false},
		{[]interface{}{"ANOTHER_COMMAND", "Test string", 1}, false},
	}

	command := Cmd{
		Name: "TEST_COMMAND",
		Args: []interface{}{"Test string", NewAnyInt()},
	}

	for pos, element := range fuzzyCommandTestInput {
		if retVal := match(element.arguments[0].(string), element.arguments[1:], &command); retVal != element.match {
			t.Fatalf("comparing fuzzy comand failed. Comparison between comand [%#v] and test arguments : [%#v] at position %v returned %v while it should have returned %v",
				command, element.arguments, pos, retVal, element.match)
		}
	}
}

func TestFuzzyCommandMatchAnyDouble(t *testing.T) {
	var fuzzyCommandTestInput = []struct {
		arguments []interface{}
		match     bool
	}{
		{[]interface{}{"TEST_COMMAND", "Test string", 1.123}, true},
		{[]interface{}{"TEST_COMMAND", "Test string", 1234567.89}, true},
		{[]interface{}{"TEST_COMMAND", 1.0, "Test string"}, false},
		{[]interface{}{"TEST_COMMAND", "Test string", 1.123, 11.22}, false},
		{[]interface{}{"TEST_COMMAND", "AnotherString", 1.1111}, false},
		{[]interface{}{"TEST_COMMAND", 1.122}, false},
		{[]interface{}{"TEST_COMMAND", "AnotherString"}, false},
		{[]interface{}{"TEST_COMMAND", "Test string", 1}, false},
		{[]interface{}{"TEST_COMMAND", "Test string", "This is not a double"}, false},
		{[]interface{}{"ANOTHER_COMMAND", "Test string", 1.123}, false},
	}

	command := Cmd{
		Name: "TEST_COMMAND",
		Args: []interface{}{"Test string", NewAnyDouble()},
	}

	for pos, element := range fuzzyCommandTestInput {
		if retVal := match(element.arguments[0].(string), element.arguments[1:], &command); retVal != element.match {
			t.Errorf("comparing fuzzy comand failed. Comparison between comand [%+v] and test arguments : [%v] at position %v returned %v while it should have returned %v",
				command, element.arguments, pos, retVal, element.match)
		}
	}
}

func TestFuzzyCommandMatchAnyData(t *testing.T) {
	var fuzzyCommandTestInput = []struct {
		arguments []interface{}
		match     bool
	}{
		{[]interface{}{"TEST_COMMAND", "Test string", "Another string"}, true},
		{[]interface{}{"TEST_COMMAND", "Test string", 12344}, true},
		{[]interface{}{"TEST_COMMAND", "Test string", func() {}}, true}, // func
		{[]interface{}{"TEST_COMMAND", "Test string", []string{"Slice of", "strings"}}, true},
		{[]interface{}{"TEST_COMMAND", "Test string", "Another string", 11.22}, false},
	}

	command := Cmd{
		Name: "TEST_COMMAND",
		Args: []interface{}{"Test string", NewAnyData()},
	}

	for pos, element := range fuzzyCommandTestInput {
		if retVal := match(element.arguments[0].(string), element.arguments[1:], &command); retVal != element.match {
			t.Errorf("comparing fuzzy comand failed. Comparison between comand [%+v] and test arguments : [%v] at position %v returned %v while it should have returned %v",
				command, element.arguments, pos, retVal, element.match)
		}
	}
}

func TestFindWithFuzzy(t *testing.T) {
	connection := NewConn()
	connection.Command("HGETALL", NewAnyInt(), NewAnyDouble(), "Test string")

	if connection.find("HGETALL", []interface{}{1, 2.0}) != nil {
		t.Error("Returning command without comparing all registered arguments")
	}

	if connection.find("HGETALL", []interface{}{1, 2.0, "Test string", "a"}) != nil {
		t.Error("Returning command without comparing all informed arguments")
	}

	if connection.find("HSETALL", []interface{}{1, 2.0, "Test string"}) != nil {
		t.Error("Returning command when the name is different")
	}

	if connection.find("HGETALL", []interface{}{1.0, "Test string", 2}) != nil {
		t.Error("Returning command with arguments in a different order")
	}

	if connection.find("HGETALL", []interface{}{1, 2.0, "Test string"}) == nil {
		t.Error("Could not find command with arguments in the same order")
	}
}

func TestRemoveRelatedFuzzyCommands(t *testing.T) {
	connection := NewConn()
	connection.Command("HGETALL", 1, 2.0, "c")                // saved , non fuzzy
	connection.Command("HGETALL", NewAnyInt(), 2.0, "c")      // saved , fuzzy
	connection.Command("HGETALL", NewAnyInt(), 2.0, "c")      // not saved!! , fuzzy
	connection.Command("HGETALL", NewAnyDouble(), 2.0, "c")   // saved , fuzzy
	connection.Command("COMMAND2", NewAnyInt(), 2.0, "c")     // saved , fuzzy
	connection.Command("HGETALL", NewAnyInt(), 5.0, "c")      // saved, fuzzy
	connection.Command("HGETALL", NewAnyInt(), 2.0, "d")      // saved, fuzzy
	connection.Command("HGETALL", NewAnyInt(), 2, "c")        // saved, fuzzy
	connection.Command("HGETALL", NewAnyInt(), 2.0, "c", "d") // saved, fuzzy
	connection.Command("HGETALL", 1, NewAnyDouble(), "c")     // saved, fuzzy

	if len(connection.commands) != 9 {
		t.Errorf("Non fuzzy command cound invalid, expected 9, got %d", len(connection.commands))
	}
}
