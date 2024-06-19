package strings

import (
    "testing"
)

var emptyString = ""
var nonEmptyString = "abc"
var flagtests = []struct {
    in *string
    out string
}{
    {nil, "a"},
    {&emptyString, ""},
    {&nonEmptyString, nonEmptyString},
}

func TestStringFromPointer(t *testing.T) {
    for _, tt := range flagtests {
        out := StringFromPointer(tt.in)

        if tt.out != out {
            t.Fatalf(`StringFromPointer(%v) == %q, expected %q.`, tt.in, out, tt.out)
        }
    }
}
