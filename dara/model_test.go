package dara

import (
	"testing"

	"github.com/alibabacloud-go/tea/utils"
)

// TestValidator 测试Validator接口实现
type TestValidator struct {
	Name  *string `json:"name"`
	Age   *int    `json:"age"`
	Email *string `json:"email"`
}

func (tv *TestValidator) Validate() error {
	if err := ValidateRequired(tv.Name, "name"); err != nil {
		return err
	}
	if tv.Name != nil {
		if err := ValidateMaxLength(*tv.Name, 10, "Name"); err != nil {
			return err
		}
		if err := ValidateMinLength(*tv.Name, 2, "Name"); err != nil {
			return err
		}
	}
	if tv.Age != nil {
		if err := ValidateMinimum(float64(*tv.Age), 0.0, "Age"); err != nil {
			return err
		}
		if err := ValidateMaximum(float64(*tv.Age), 150.0, "Age"); err != nil {
			return err
		}
	}
	if tv.Email != nil {
		if err := ValidatePattern(*tv.Email, "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}", "Email"); err != nil {
			return err
		}
	}
	return nil
}

func TestValidatorInterface(t *testing.T) {
	// 测试成功的情况
	name := "Alice"
	age := 25
	email := "alice@example.com"

	validator := &TestValidator{
		Name:  &name,
		Age:   &age,
		Email: &email,
	}

	err := validator.Validate()
	utils.AssertNil(t, err)

	// 测试必填字段为空的情况
	validator2 := &TestValidator{
		Name: nil,
		Age:  &age,
	}

	err = validator2.Validate()
	utils.AssertEqual(t, "name should be setted", err.Error())

	// 测试长度超限的情况
	longName := "ThisNameIsTooLong"
	validator3 := &TestValidator{
		Name: &longName,
		Age:  &age,
	}

	err = validator3.Validate()
	utils.AssertEqual(t, "The length of Name is 17 which is more than 10", err.Error())
}

func TestValidateRequired(t *testing.T) {
	// 测试非空值
	value := "test"
	err := ValidateRequired(&value, "testField")
	utils.AssertNil(t, err)

	// 测试nil值
	err = ValidateRequired(nil, "testField")
	utils.AssertEqual(t, "testField should be setted", err.Error())

	// 测试指针为nil的情况
	var nilPtr *string
	err = ValidateRequired(nilPtr, "testField")
	utils.AssertEqual(t, "testField should be setted", err.Error())
}

func TestValidateMaxLength(t *testing.T) {
	// 测试字符串长度验证
	shortStr := "abc"
	err := ValidateMaxLength(shortStr, 5, "testField")
	utils.AssertNil(t, err)

	longStr := "abcdefghijk"
	err = ValidateMaxLength(longStr, 5, "testField")
	utils.AssertEqual(t, "The length of testField is 11 which is more than 5", err.Error())

	// 测试字符串指针
	str := "test"
	err = ValidateMaxLength(&str, 5, "testField")
	utils.AssertNil(t, err)

	// 测试Unicode字符串
	unicodeStr := "测试"
	err = ValidateMaxLength(unicodeStr, 5, "testField")
	utils.AssertNil(t, err)

	longUnicodeStr := "这是一个很长的测试字符串"
	err = ValidateMaxLength(longUnicodeStr, 5, "testField")
	utils.AssertEqual(t, "The length of testField is 12 which is more than 5", err.Error())

	// 测试切片长度
	slice := []string{"a", "b", "c"}
	err = ValidateMaxLength(slice, 5, "testField")
	utils.AssertNil(t, err)

	longSlice := []string{"a", "b", "c", "d", "e", "f"}
	err = ValidateMaxLength(longSlice, 3, "testField")
	utils.AssertEqual(t, "The length of testField is 6 which is more than 3", err.Error())
}

func TestValidateMinLength(t *testing.T) {
	// 测试字符串长度验证
	str := "abcde"
	err := ValidateMinLength(str, 3, "testField")
	utils.AssertNil(t, err)

	shortStr := "ab"
	err = ValidateMinLength(shortStr, 5, "testField")
	utils.AssertEqual(t, "The length of testField is 2 which is less than 5", err.Error())

	// 测试空字符串
	emptyStr := ""
	err = ValidateMinLength(emptyStr, 1, "testField")
	utils.AssertEqual(t, "The length of testField is 0 which is less than 1", err.Error())

	// 测试字符串指针
	testStr := "hello"
	err = ValidateMinLength(&testStr, 3, "testField")
	utils.AssertNil(t, err)

	// 测试nil指针
	var nilPtr *string
	err = ValidateMinLength(nilPtr, 1, "testField")
	utils.AssertEqual(t, "The length of testField is 0 which is less than 1", err.Error())
}

func TestValidatePattern(t *testing.T) {
	// 测试匹配的模式
	email := "test@example.com"
	err := ValidatePattern(email, "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}", "email")
	utils.AssertNil(t, err)

	// 测试不匹配的模式
	invalidEmail := "invalid-email"
	err = ValidatePattern(invalidEmail, "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}", "email")
	utils.AssertEqual(t, "invalid-email is not matched [a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}", err.Error())

	// 测试空字符串
	emptyStr := ""
	err = ValidatePattern(emptyStr, "[a-z]+", "testField")
	utils.AssertNil(t, err) // 空字符串应该跳过验证

	// 测试字符串指针
	pattern := "abc"
	err = ValidatePattern(&pattern, "[a-d]*", "testField")
	utils.AssertNil(t, err)

	// 测试无效的正则表达式
	err = ValidatePattern("test", "[", "testField")
	utils.AssertNotNil(t, err) // 应该返回正则编译错误
}

func TestValidateMaximum(t *testing.T) {
	// 测试int类型
	intVal := 5
	err := ValidateMaximum(intVal, 10.0, "testField")
	utils.AssertNil(t, err)

	largeIntVal := 15
	err = ValidateMaximum(largeIntVal, 10.0, "testField")
	utils.AssertEqual(t, "The size of testField is 15.000000 which is greater than 10.000000", err.Error())

	// 测试int指针
	val := 8
	err = ValidateMaximum(&val, 10.0, "testField")
	utils.AssertNil(t, err)

	// 测试float类型
	floatVal := 3.14
	err = ValidateMaximum(floatVal, 5.0, "testField")
	utils.AssertNil(t, err)

	// 测试各种数值类型
	var int8Val int8 = 100
	err = ValidateMaximum(int8Val, 127.0, "testField")
	utils.AssertNil(t, err)

	var uint16Val uint16 = 1000
	err = ValidateMaximum(uint16Val, 2000.0, "testField")
	utils.AssertNil(t, err)

	// 测试非数值类型
	str := "not a number"
	err = ValidateMaximum(str, 10.0, "testField")
	utils.AssertNil(t, err) // 非数值类型应该跳过验证
}

func TestValidateMinimum(t *testing.T) {
	// 测试int类型
	intVal := 15
	err := ValidateMinimum(intVal, 10.0, "testField")
	utils.AssertNil(t, err)

	smallIntVal := 5
	err = ValidateMinimum(smallIntVal, 10.0, "testField")
	utils.AssertEqual(t, "The size of testField is 5.000000 which is less than 10.000000", err.Error())

	// 测试负数
	negativeVal := -5
	err = ValidateMinimum(negativeVal, 0.0, "testField")
	utils.AssertEqual(t, "The size of testField is -5.000000 which is less than 0.000000", err.Error())

	// 测试零值
	zeroVal := 0
	err = ValidateMinimum(zeroVal, 0.0, "testField")
	utils.AssertNil(t, err)

	// 测试float指针
	floatVal := 2.5
	err = ValidateMinimum(&floatVal, 1.0, "testField")
	utils.AssertNil(t, err)

	// 测试nil指针
	var nilPtr *int
	err = ValidateMinimum(nilPtr, 1.0, "testField")
	utils.AssertNil(t, err) // nil指针应该跳过验证
}

func TestValidateArray(t *testing.T) {
	// 测试字符串切片
	strSlice := []*string{String("a"), String("b"), String("c")}
	validator := func(item interface{}) error {
		if str, ok := item.(*string); ok && str != nil {
			return ValidateMaxLength(*str, 5, "item")
		}
		return nil
	}

	err := ValidateArray(strSlice, validator)
	utils.AssertNil(t, err)

	// 测试验证失败的情况
	longStrSlice := []*string{String("toolong")}
	err = ValidateArray(longStrSlice, func(item interface{}) error {
		if str, ok := item.(*string); ok && str != nil {
			return ValidateMaxLength(*str, 3, "item")
		}
		return nil
	})
	utils.AssertEqual(t, "The length of item is 7 which is more than 3", err.Error())

	// 测试int切片
	intSlice := []*int{Int(1), Int(2), Int(3)}
	err = ValidateArray(intSlice, func(item interface{}) error {
		if val, ok := item.(*int); ok && val != nil {
			return ValidateMaximum(*val, 5.0, "item")
		}
		return nil
	})
	utils.AssertNil(t, err)
}

func TestValidateMap(t *testing.T) {
	// 测试字符串映射
	strMap := map[string]*string{
		"key1": String("value1"),
		"key2": String("value2"),
	}

	validator := func(value interface{}) error {
		if str, ok := value.(*string); ok && str != nil {
			return ValidateMaxLength(*str, 10, "value")
		}
		return nil
	}

	err := ValidateMap(strMap, validator)
	utils.AssertNil(t, err)

	// 测试验证失败的情况
	longValueMap := map[string]*string{
		"key1": String("verylongvalue"),
	}

	err = ValidateMap(longValueMap, func(value interface{}) error {
		if str, ok := value.(*string); ok && str != nil {
			return ValidateMaxLength(*str, 5, "value")
		}
		return nil
	})
	utils.AssertEqual(t, "The length of value is 13 which is more than 5", err.Error())
}

func TestGetStringValue(t *testing.T) {
	// 测试直接字符串
	str := getStringValue("test")
	utils.AssertEqual(t, "test", str)

	// 测试字符串指针
	testStr := "hello"
	str = getStringValue(&testStr)
	utils.AssertEqual(t, "hello", str)

	// 测试nil指针
	var nilPtr *string
	str = getStringValue(nilPtr)
	utils.AssertEqual(t, "", str)

	// 测试非字符串类型
	str = getStringValue(123)
	utils.AssertEqual(t, "", str)
}

func TestGetNumericValue(t *testing.T) {
	// 测试各种数值类型
	tests := []struct {
		input    interface{}
		expected float64
		valid    bool
	}{
		{int(42), 42.0, true},
		{int8(127), 127.0, true},
		{int16(32767), 32767.0, true},
		{int32(2147483647), 2147483647.0, true},
		{int64(9223372036854775807), 9223372036854775807.0, true},
		{uint(42), 42.0, true},
		{uint8(255), 255.0, true},
		{uint16(65535), 65535.0, true},
		{uint32(4294967295), 4294967295.0, true},
		{uint64(18446744073709551615), 18446744073709551615.0, true},
		{float32(3.14), 3.140000104904175, true}, // float32精度损失
		{float64(3.14159), 3.14159, true},
		{"not a number", 0, false},
		{nil, 0, false},
	}

	for _, test := range tests {
		value, valid := getNumericValue(test.input)
		utils.AssertEqual(t, test.valid, valid)
		if valid {
			utils.AssertEqual(t, test.expected, value)
		}
	}

	// 测试指针类型
	intVal := 42
	value, valid := getNumericValue(&intVal)
	utils.AssertEqual(t, true, valid)
	utils.AssertEqual(t, 42.0, value)

	// 测试nil指针
	var nilPtr *int
	value, valid = getNumericValue(nilPtr)
	utils.AssertEqual(t, false, valid)
	utils.AssertEqual(t, 0.0, value)
}

func TestGetValueLength(t *testing.T) {
	// 测试字符串长度
	length := getValueLength("hello")
	utils.AssertEqual(t, 5, length)

	// 测试Unicode字符串
	length = getValueLength("你好")
	utils.AssertEqual(t, 2, length)

	// 测试字符串指针
	str := "test"
	length = getValueLength(&str)
	utils.AssertEqual(t, 4, length)

	// 测试nil字符串指针
	var nilStr *string
	length = getValueLength(nilStr)
	utils.AssertEqual(t, 0, length)

	// 测试切片
	slice := []string{"a", "b", "c"}
	length = getValueLength(slice)
	utils.AssertEqual(t, 3, length)

	// 测试指针切片
	ptrSlice := []*string{&str, &str}
	length = getValueLength(ptrSlice)
	utils.AssertEqual(t, 2, length)

	// 测试映射
	m := map[string]string{"a": "1", "b": "2"}
	length = getValueLength(m)
	utils.AssertEqual(t, 2, length)

	// 测试不支持的类型
	length = getValueLength(123)
	utils.AssertEqual(t, 0, length)
}

// 测试向后兼容性，确保没有实现Validator接口的结构体仍然使用反射版本
type NonValidatorStruct struct {
	Name *string `json:"name,omitempty" require:"true" maxLength:"10"`
	Age  *int    `json:"age,omitempty" minimum:"0" maximum:"150"`
}

func TestBackwardCompatibility(t *testing.T) {
	name := "Alice"
	age := 25

	// 测试成功情况
	nonValidator := &NonValidatorStruct{
		Name: &name,
		Age:  &age,
	}

	err := Validate(nonValidator)
	utils.AssertNil(t, err)

	// 测试必填验证
	nonValidator = &NonValidatorStruct{
		Name: nil,
		Age:  &age,
	}

	err = Validate(nonValidator)
	utils.AssertEqual(t, "name should be setted", err.Error())

	// 测试长度验证
	longName := "ThisNameIsTooLong"
	nonValidator = &NonValidatorStruct{
		Name: &longName,
		Age:  &age,
	}

	err = Validate(nonValidator)
	utils.AssertEqual(t, "The length of Name is 17 which is more than 10", err.Error())
}

// 性能基准测试
func BenchmarkValidatorInterface(b *testing.B) {
	name := "Alice"
	age := 25
	email := "alice@example.com"

	validator := &TestValidator{
		Name:  &name,
		Age:   &age,
		Email: &email,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Validate(validator)
	}
}

func BenchmarkReflectionValidation(b *testing.B) {
	name := "Alice"
	age := 25

	nonValidator := &NonValidatorStruct{
		Name: &name,
		Age:  &age,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Validate(nonValidator)
	}
}
