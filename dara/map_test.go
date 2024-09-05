package dara

import (
	"reflect"
	"testing"
)

type Person struct {
	Name string
	Age  int
}

func TestEntries(t *testing.T) {
	// 定义一个包含多种类型的 map
	type Person struct {
		Name string
		Age  int
	}

	testMap := map[string]interface{}{
		"one":   1,
		"two":   "two",
		"three": &Person{Name: "Alice", Age: 30},
	}

	entries := Entries(testMap)

	if len(entries) != 3 {
		t.Errorf("expected %d entries, got %d", 3, len(entries))
	}

	for _, entry := range entries {
		if !reflect.DeepEqual(entry.Value, testMap[entry.Key]) {
			t.Errorf("expected entry %s to be %v, got %v", entry.Key, testMap[entry.Key], entry.Value)
		}
	}
}

func TestKeySet(t *testing.T) {
	testMap := map[string]interface{}{
		"one":   1,
		"two":   "two",
		"three": &Person{Name: "Alice", Age: 30},
	}

	keys := KeySet(testMap)
	str1, str2, str3 := "one", "two", "three"
	expectedKeys := []*string{&str1, &str2, &str3}

	if len(keys) != len(expectedKeys) {
		t.Errorf("expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for _, key := range keys {
		if !ArrContains(expectedKeys, key) {
			t.Errorf("expected key %s to be in the array %v, but not", key, expectedKeys)
		}
	}
}
