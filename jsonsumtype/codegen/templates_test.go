package codegen

import "testing"

func TestQuoteSingle(t *testing.T) {
	testCases := []struct {
		input, expected string
	}{
		{`hello`, `'hello'`},
		{`'hello'`, `'\'hello\''`},
		{`"hello"`, `'"hello"'`},
		{`It's a string`, `'It\'s a string'`},
		{``, `''`},
		{`abc"xyz`, `'abc"xyz'`},
		{`abc'xyz`, `'abc\'xyz'`},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got := quoteSingle(tc.input)
			if got != tc.expected {
				t.Errorf("quoteSingle(%s) = %s; want %s",
					tc.input, got, tc.expected,
				)
			}
		})
	}
}
