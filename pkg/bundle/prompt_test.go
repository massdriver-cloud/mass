package bundle

import (
	"strings"
	"testing"
)

func TestNameValidate(t *testing.T) {
	goodValues := []string{
		"ab", "abc", "a1", "a-1", "a--1--2--b",
		strings.Repeat("a", 53),
	}
	for _, val := range goodValues {
		if err := bundleNameValidate(val); err != nil {
			t.Errorf("expected no error for '%s': %v", val, err)
		}
	}

	badValues := []string{
		"", "A", "ABC", "aBc", "A1", "A-1", "1-A",
		"-", "a-", "-a", "1-", "-1",
		"_", "a_", "_a", "a_b", "1_", "_1", "1_2",
		".", "a.", ".a", "a.b", "1.", ".1", "1.2",
		" ", "a ", " a", "a b", "1 ", " 1", "1 2",
		"1111", "1-1-1", "----", strings.Repeat("a", 54),
	}
	for _, val := range badValues {
		if err := bundleNameValidate(val); err == nil {
			t.Errorf("expected error for '%s'", val)
		}
	}
}

func TestConnNameValidate(t *testing.T) {
	goodValues := []string{
		"ab", "abc", "a1", "a_1", "a__1__2__b",
		strings.Repeat("a", 53),
	}
	for _, val := range goodValues {
		if err := connNameValidate(val); err != nil {
			t.Errorf("expected no error for '%s': %v", val, err)
		}
	}

	badValues := []string{
		"", "A", "ABC", "aBc", "A1", "A-1", "1-A",
		"-", "a-", "-a", "1-", "-1",
		"_", "a_", "_a", "a-b", "1_", "_1", "1_2",
		".", "a.", ".a", "a.b", "1.", ".1", "1.2",
		" ", "a ", " a", "a b", "1 ", " 1", "1 2",
		"1111", "1-1-1", "----", strings.Repeat("a", 54),
	}
	for _, val := range badValues {
		if err := connNameValidate(val); err == nil {
			t.Errorf("expected error for '%s'", val)
		}
	}
}
