package strings

import (
    "testing"
)

func TestStringFromPointerNil(t *testing.T) {
    var ptr *string
    res := StringFromPointer(ptr)
    if res != "" {
        t.Fatalf(`StringFromPointer(nil) == %q, expected empty string.`, res)
    }
}

func TestStringFromPointerToEmpty(t *testing.T) {
    input := ""
    res := StringFromPointer(&input)
    if res != "" {
        t.Fatalf(`StringFromPointer with empty string returned %q, expected empty string.`, res)
    }
}
