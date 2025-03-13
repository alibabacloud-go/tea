package dara

import (
	"reflect"
	"testing"
)

type MyStruct struct {
	Name string
}

func TestArrContains(t *testing.T) {
	// Create test data
	str1 := "Hello"
	str2 := "World"
	ptrStrArr := []*string{&str1, &str2}

	// Test with string pointer array
	if !ArrContains(ptrStrArr, "World") {
		t.Errorf("Expected true, but got false")
	}

	if ArrContains(ptrStrArr, "Go") {
		t.Errorf("Expected false, but got true")
	}

	// Create integer pointer values
	num1 := 1
	num2 := 2
	ptrIntArr := []*int{&num1, &num2}

	// Test with integer pointer array
	if !ArrContains(ptrIntArr, 2) {
		t.Errorf("Expected true, but got false")
	}

	if ArrContains(ptrIntArr, 3) {
		t.Errorf("Expected false, but got true")
	}

	// Create struct pointers
	struct1 := &MyStruct{Name: "One"}
	struct2 := &MyStruct{Name: "Two"}
	structPtrArr := []*MyStruct{struct1, struct2}

	// Test struct pointer array
	if ArrContains(structPtrArr, &MyStruct{Name: "One"}) {
		t.Errorf("Expected false, but got true")
	}

	// Check for existence by value
	if ArrContains(structPtrArr, "One") {
		t.Errorf("Expected false, but got true")
	}

	if !ArrContains(structPtrArr, struct1) {
		t.Errorf("Expected true, but got false")
	}

	interfaceArr := []interface{}{str1, num1, struct1}

	if !ArrContains(interfaceArr, "Hello") {
		t.Errorf("Expected true, but got false")
	}

	if ArrContains(interfaceArr, "World") {
		t.Errorf("Expected false, but got true")
	}

	if !ArrContains(interfaceArr, 1) {
		t.Errorf("Expected true, but got false")
	}

	if ArrContains(interfaceArr, 2) {
		t.Errorf("Expected false, but got true")
	}

	if ArrContains(interfaceArr, &MyStruct{Name: "One"}) {
		t.Errorf("Expected false, but got true")
	}

	if !ArrContains(interfaceArr, struct1) {
		t.Errorf("Expected true, but got false")
	}
}

func TestArrIndex(t *testing.T) {
	// Create test data for string pointer array
	str1 := "Hello"
	str2 := "World"
	ptrStrArr := []*string{&str1, &str2}

	// Test with string pointer array
	if index := ArrIndex(ptrStrArr, "World"); index != 1 {
		t.Errorf("Expected index 1, but got %d", index)
	}

	if index := ArrIndex(ptrStrArr, "Go"); index != -1 {
		t.Errorf("Expected index -1, but got %d", index)
	}

	// Create integer pointer values
	num1 := 1
	num2 := 2
	ptrIntArr := []*int{&num1, &num2}

	// Test with integer pointer array
	if index := ArrIndex(ptrIntArr, 2); index != 1 {
		t.Errorf("Expected index 1, but got %d", index)
	}

	if index := ArrIndex(ptrIntArr, 3); index != -1 {
		t.Errorf("Expected index -1, but got %d", index)
	}

	// Create struct pointers
	struct1 := &MyStruct{Name: "One"}
	struct2 := &MyStruct{Name: "Two"}
	structPtrArr := []*MyStruct{struct1, struct2}

	// Test struct pointer array
	if index := ArrIndex(structPtrArr, &MyStruct{Name: "One"}); index != -1 {
		t.Errorf("Expected index -1, but got %d", index)
	}

	interfaceArr := []interface{}{str1, num1, struct1}

	if index := ArrIndex(interfaceArr, 1); index != 1 {
		t.Errorf("Expected index 1, but got %d", index)
	}

	if index := ArrIndex(interfaceArr, "Hello"); index != 0 {
		t.Errorf("Expected index 0, but got %d", index)
	}

	if index := ArrIndex(interfaceArr, struct1); index != 2 {
		t.Errorf("Expected index 2, but got %d", index)
	}
}

func TestArrJoin(t *testing.T) {
	// Create test data
	str1 := "Hello"
	str2 := "World"
	ptrStrArr := []*string{&str1, &str2}

	// Test joining strings
	result := ArrJoin(ptrStrArr, ", ")
	expected := "Hello, World"
	if result != expected {
		t.Errorf("Expected '%s', but got '%s'", expected, result)
	}

	// Create integer pointer values
	num1 := 1
	num2 := 2
	ptrIntArr := []*int{&num1, &num2}

	// Test joining integers
	result = ArrJoin(ptrIntArr, " + ")
	expected = "1 + 2"
	if result != expected {
		t.Errorf("Expected '%s', but got '%s'", expected, result)
	}

	// Create mixed types (if needed)
	struct1 := &MyStruct{Name: "One"}
	ptrMixedArr := []interface{}{str1, num1, struct1}

	// Test joining mixed types
	result = ArrJoin(ptrMixedArr, " | ")
	expected = "Hello | 1"
	if result != expected {
		t.Errorf("Expected '%s', but got '%s'", expected, result)
	}
}

func TestArrShift(t *testing.T) {
	// Create test data for string pointer array
	str1 := "Hello"
	str2 := "World"
	ptrStrArr := []*string{&str1, &str2}

	// Test shifting strings
	removed := ArrShift(&ptrStrArr)
	if removed != &str1 {
		t.Errorf("Expected '%v', but got '%v'", &str1, removed)
	}

	// After shifting, the first element should now be "World"
	if ptrStrArr[0] != &str2 {
		t.Errorf("Expected next element to be '%v', but got '%v'", &str2, ptrStrArr[0])
	}

	// Create integer pointer values
	num1 := 1
	num2 := 2
	ptrIntArr := []*int{&num1, &num2}

	// Test shifting integers
	removedInt := ArrShift(&ptrIntArr)
	if removedInt != &num1 {
		t.Errorf("Expected '%v', but got '%v'", &num1, removedInt)
	}

	// Create struct pointers
	struct1 := &MyStruct{Name: "One"}
	struct2 := &MyStruct{Name: "Two"}
	structPtrArr := []*MyStruct{struct1, struct2}

	// Test struct pointer array
	removedStruct := ArrShift(&structPtrArr)
	if removedStruct != struct1 {
		t.Errorf("Expected '%v', but got '%v'", struct1, removedStruct)
	}

	interfaceArr := []interface{}{str1, num1, struct1}

	removedStr := ArrShift(&interfaceArr)

	if removedStr != str1 {
		t.Errorf("Expected '%v', but got '%v'", str1, removedStr)
	}

	removedInt = ArrShift(&interfaceArr)

	if removedInt != num1 {
		t.Errorf("Expected '%v', but got '%v'", removedInt, num1)
	}

	if interfaceArr[0] != struct1 {
		t.Errorf("Expected next element to be '%v', but got '%v'", struct1, interfaceArr[0])
	}
}

func TestArrPop(t *testing.T) {
	// Create test data for string pointer array
	str1 := "Hello"
	str2 := "World"
	ptrStrArr := []*string{&str1, &str2}

	// Test popping strings
	removed := ArrPop(&ptrStrArr)
	if removed != &str2 {
		t.Errorf("Expected '%v', but got '%v'", &str2, removed)
	}

	// After popping, the array should only contain "Hello"
	if len(ptrStrArr) != 1 || ptrStrArr[0] != &str1 {
		t.Errorf("Expected remaining element to be '%v', but got '%v'", &str1, ptrStrArr)
	}

	// Create integer pointer values
	num1 := 1
	num2 := 2
	ptrIntArr := []*int{&num1, &num2}

	// Test popping integers
	removedInt := ArrPop(&ptrIntArr)
	if removedInt != &num2 {
		t.Errorf("Expected '%v', but got '%v'", &num2, removedInt)
	}

	// Create struct pointers
	struct1 := &MyStruct{Name: "One"}
	struct2 := &MyStruct{Name: "Two"}
	structPtrArr := []*MyStruct{struct1, struct2}

	// Test struct pointer array
	removedStruct := ArrPop(&structPtrArr)
	if removedStruct != struct2 {
		t.Errorf("Expected '%v', but got '%v'", struct2, removedStruct)
	}
}

func TestArrUnshift(t *testing.T) {
	// Create test data for string pointer array
	str1 := "World"
	ptrStrArr := []*string{&str1}

	// New string to be added
	str2 := "Hello"

	// Test unshifting strings
	length := ArrUnshift(&ptrStrArr, &str2)
	if length != 2 {
		t.Fatalf("Expected  array length is 2 but %d", length)
	}

	if ptrStrArr[0] != &str2 || ptrStrArr[1] != &str1 {
		t.Errorf("Expected '%v', '%v', but got '%v'", &str2, &str1, ptrStrArr)
	}

	// Test unshifting integers
	num1 := 2
	ptrIntArr := []*int{&num1}
	num2 := 1

	length = ArrUnshift(&ptrIntArr, &num2)
	if length != 2 {
		t.Fatalf("Expected  array length is 2 but %d", length)
	}

	if ptrIntArr[0] != &num2 || ptrIntArr[1] != &num1 {
		t.Errorf("Expected '%v', '%v', but got '%v'", &num2, &num1, ptrIntArr)
	}

	ptrMixedArr := []interface{}{str1, num1}
	struct1 := &MyStruct{Name: "One"}

	length = ArrUnshift(&ptrMixedArr, struct1)

	if length != 3 {
		t.Fatalf("Expected  array length is 3 but %d", length)
	}

	if ptrMixedArr[0] != struct1 {
		t.Errorf("Expected ptrMixedArr index 2 is '%v', but got '%v'", struct1, ptrMixedArr[0])
	}
}

func TestArrPush(t *testing.T) {
	// Create test data for string pointer array
	str1 := "Hello"
	ptrStrArr := []*string{&str1}

	// New string to be added
	str2 := "World"

	// Test pushing strings
	length := ArrPush(&ptrStrArr, &str2)
	if length != 2 {
		t.Fatalf("Expected  array length is 2 but %d", length)
	}

	if ptrStrArr[0] != &str1 || ptrStrArr[1] != &str2 {
		t.Errorf("Expected '%v', '%v', but got '%v'", &str1, &str2, ptrStrArr)
	}

	// Test pushing integers
	num1 := 1
	ptrIntArr := []*int{&num1}
	num2 := 2

	length = ArrPush(&ptrIntArr, &num2)
	if length != 2 {
		t.Fatalf("Expected  array length is 2 but %d", length)
	}

	if ptrIntArr[0] != &num1 || ptrIntArr[1] != &num2 {
		t.Errorf("Expected '%v', '%v', but got '%v'", &num1, &num2, ptrIntArr)
	}

	ptrMixedArr := []interface{}{str1, num1}
	struct1 := &MyStruct{Name: "One"}

	length = ArrPush(&ptrMixedArr, struct1)

	if length != 3 {
		t.Fatalf("Expected  array length is 3 but %d", length)
	}

	if ptrMixedArr[2] != struct1 {
		t.Errorf("Expected ptrMixedArr index 2 is '%v', but got '%v'", struct1, ptrMixedArr[2])
	}
}

func TestConcatArr(t *testing.T) {
	str1, str2, str3, str4 := "A", "B", "C", "D"
	// String arrays
	strArr1 := []*string{&str1, &str2}
	strArr2 := []*string{&str3, &str4}

	// Test concatenating string arrays
	result := ConcatArr(strArr1, strArr2)
	expected := []*string{&str1, &str2, &str3, &str4}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected '%v', but got '%v'", expected, result)
	}

	num1, num2, num3, num4 := 1, 2, 3, 4
	// Integer arrays
	intArr1 := []*int{&num1, &num2}
	intArr2 := []*int{&num3, &num4}

	// Test concatenating integer arrays
	result = ConcatArr(intArr1, intArr2)
	expectedInts := []*int{&num1, &num2, &num3, &num4}

	if !reflect.DeepEqual(result, expectedInts) {
		t.Errorf("Expected '%v', but got '%v'", expectedInts, result)
	}

	// Mixed type arrays
	mixedArr1 := []interface{}{1, "two"}
	mixedArr2 := []interface{}{3.0, "four"}

	// Test concatenating mixed type arrays
	result = ConcatArr(mixedArr1, mixedArr2)
	expectedMixed := []interface{}{1, "two", 3.0, "four"}

	if !reflect.DeepEqual(result, expectedMixed) {
		t.Errorf("Expected '%v', but got '%v'", expectedMixed, result)
	}
}

func TestArrAppend(t *testing.T) {
	// 测试用例1：插入中间位置
	t.Run("Append to middle of an array", func(t *testing.T) {
		numbers := []*int{new(int), new(int), new(int)}
		for i := range numbers {
			*numbers[i] = i + 1
		}

		// 将 9 插入到索引 1
		valueToInsert := new(int)
		*valueToInsert = 9

		// 期望的结果
		expected := []*int{new(int), new(int), new(int), new(int)}
		*expected[0], *expected[1], *expected[2], *expected[3] = 1, 9, 2, 3

		defer func() {
			if !reflect.DeepEqual(numbers, expected) {
				t.Errorf("Expected %+v, but got %+v", expected, numbers)
			}
		}()

		ArrAppend(&numbers, valueToInsert, 1)
	})

	// 测试用例2: 尝试在越界处插入
	t.Run("Index out of bounds", func(t *testing.T) {
		numbers := []*int{new(int), new(int), new(int)}
		for i := range numbers {
			*numbers[i] = i + 1
		}

		defer func() {
			// Defer 检查：越界情况下，数组应保持不变
			expected := []*int{new(int), new(int), new(int)}
			*expected[0], *expected[1], *expected[2] = 1, 2, 3
			if !reflect.DeepEqual(numbers, expected) {
				t.Errorf("Index out of bounds should not modify array, but got %+v", numbers)
			}
		}()

		valueToInsert := new(int)
		*valueToInsert = 9
		ArrAppend(&numbers, valueToInsert, 10) // 超出范围
	})
}


func TestArrRemove(t *testing.T) {
	// Create test data for string pointer array
	str1 := "A"
	str2 := "B"
	str3 := "C"
	ptrStrArr := []*string{&str1, &str2, &str3}

	// Test removing an element in the middle
	ArrRemove(&ptrStrArr, "B")

	if len(ptrStrArr) != 2 || ptrStrArr[0] != &str1 || ptrStrArr[1] != &str3 {
		t.Errorf("Expected '%v', '%v', but got '%v'", &str1, &str3, ptrStrArr)
	}

	// Test removing the first element
	ArrRemove(&ptrStrArr, "A")
	if len(ptrStrArr) != 1 || ptrStrArr[0] != &str3 {
		t.Errorf("Expected '%v', but got '%v'", &str3, ptrStrArr)
	}

	num1 := 1
	num2 := 2
	num3 := 3
	ptrIntArr := []*int{&num1, &num2, &num3}

	ArrRemove(&ptrIntArr, 2)

	if len(ptrIntArr) != 2 || ptrIntArr[0] != &num1 || ptrIntArr[1] != &num3 {
		t.Errorf("Expected '%v', '%v', but got '%v'", &num1, &num3, ptrIntArr)
	}

	// Test removing the first element
	ArrRemove(&ptrIntArr, 3)
	if len(ptrIntArr) != 1 || ptrIntArr[0] != &num1 {
		t.Errorf("Expected '%v', but got '%v'", &num1, ptrIntArr)
	}

	struct1 := &MyStruct{Name: "One"}
	struct2 := &MyStruct{Name: "Two"}
	struct3 := &MyStruct{Name: "Three"}
	structPtrArr := []*MyStruct{struct1, struct2, struct3}

	ArrRemove(&structPtrArr, struct2)
	if len(structPtrArr) != 2 || structPtrArr[0] != struct1 || structPtrArr[1] != struct3 {
		t.Errorf("Expected '%v', '%v', but got '%v'", struct1, struct3, structPtrArr)
	}

	interfaceArr := []interface{}{str1, num1, struct1}

	ArrRemove(&interfaceArr, num1)
	if len(interfaceArr) != 2 || interfaceArr[0] != str1 || interfaceArr[1] != struct1 {
		t.Errorf("Expected '%s', '%v', but got '%v'", str1, struct1, interfaceArr)
	}

	ArrRemove(&interfaceArr, struct1)
	if len(interfaceArr) != 1 || interfaceArr[0] != str1 {
		t.Errorf("Expected '%s', but got '%v'", str1, interfaceArr)
	}
}
func TestSortArrIntPtr(t *testing.T) {
	num1, num2, num3, num4 := 5, 3, 4, 6
	intPtrArr := []*int{&num1, &num2, &num3, &num4}
	sortedAsc := SortArr(intPtrArr, "ASC").([]*int) // 获取排序结果
	if !reflect.DeepEqual([]int{*sortedAsc[0], *sortedAsc[1], *sortedAsc[2]}, []int{3, 4, 5}) {
		t.Errorf("Ascending sort failed, expected: %v, but got %v", []*int{&num1, &num3, &num2}, sortedAsc)
	}
	sortedDesc := SortArr(intPtrArr, "DESC").([]*int) // 获取排序结果
	if !reflect.DeepEqual([]int{*sortedDesc[0], *sortedDesc[1], *sortedDesc[2]}, []int{6, 5, 4}) {
		t.Errorf("Descending sort failed, got %v", sortedDesc)
	}
}

// TestSortArrStrPtr tests the SortArr function with an array of string pointers
func TestSortArrStrPtr(t *testing.T) {
	str1, str2, str3 := "banana", "apple", "cherry"
	strPtrArr := []*string{&str1, &str2, &str3}
	sortedAsc := SortArr(strPtrArr, "AsC").([]*string) // 获取排序结果
	if !reflect.DeepEqual([]string{*sortedAsc[0], *sortedAsc[1], *sortedAsc[2]}, []string{"apple", "banana", "cherry"}) {
		t.Errorf("Ascending sort failed, got %v", sortedAsc)
	}
	sortedDesc := SortArr(strPtrArr, "dEsC").([]*string) // 获取排序结果
	if !reflect.DeepEqual([]string{*sortedDesc[0], *sortedDesc[1], *sortedDesc[2]}, []string{"cherry", "banana", "apple"}) {
		t.Errorf("Descending sort failed, got %v", sortedDesc)
	}
}

// TestSortArrStructPtr tests the SortArr function with an array of struct pointers
func TestSortArrStructPtr(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	person1 := &Person{"Alice", 30}
	person2 := &Person{"Bob", 25}
	person3 := &Person{"Charlie", 35}
	personArr := []*Person{person2, person1, person3}
	// Ascending sort by Age (first field)
	sortedAsc := SortArr(personArr, "aSc").([]*Person) // 获取排序结果
	expectedAsc := []*Person{person1, person2, person3}
	for i, p := range expectedAsc {
		if sortedAsc[i].Name != p.Name || sortedAsc[i].Age != p.Age {
			t.Errorf("Ascending sort failed, got %v", sortedAsc)
		}
	}
	// Descending sort by Age (first field)
	sortedDesc := SortArr(personArr, "DEsc").([]*Person) // 获取排序结果
	expectedDesc := []*Person{person3, person2, person1}
	for i, p := range expectedDesc {
		if sortedDesc[i].Name != p.Name || sortedDesc[i].Age != p.Age {
			t.Errorf("Descending sort failed, got %v", sortedDesc)
		}
	}
}

// TestSortArrInterface tests the SortArr function with an array of mixed interface{} types
func TestSortArrInterface(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	person1 := &Person{"Alice", 30}
	person2 := &Person{"Bob", 25}
	str1 := "banana"
	str2 := "apple"
	num1 := 5
	num2 := 3
	interfaceArr := []interface{}{str1, num1, person1, str2, num2, person2}
	sortedAsc := SortArr(interfaceArr, "asc").([]interface{}) // 获取排序结果
	expectedAsc := []interface{}{num2, num1, str2, str1, person1, person2}
	for i, v := range expectedAsc {
		if !reflect.DeepEqual(sortedAsc[i], v) {
			t.Errorf("Ascending sort failed, got %v", sortedAsc)
		}
	}
	sortedDesc := SortArr(interfaceArr, "desc").([]interface{}) // 获取排序结果
	expectedDesc := []interface{}{person2, person1, str1, str2, num1, num2}
	for i, v := range expectedDesc {
		if !reflect.DeepEqual(sortedDesc[i], v) {
			t.Errorf("Descending sort failed, got %v", sortedDesc)
		}
	}
}
