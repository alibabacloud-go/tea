package tea

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/utils"
)

func Test_Trans(t *testing.T) {
	str := String("tea")
	strVal := StringValue(str)
	utils.AssertEqual(t, "tea", strVal)
	utils.AssertEqual(t, "", StringValue(nil))

	strSlice := StringSlice([]string{"tea"})
	strSliceVal := StringSliceValue(strSlice)
	utils.AssertEqual(t, []string{"tea"}, strSliceVal)
	utils.AssertNil(t, StringSlice(nil))
	utils.AssertNil(t, StringSliceValue(nil))

	b := Bool(true)
	bVal := BoolValue(b)
	utils.AssertEqual(t, true, bVal)
	utils.AssertEqual(t, false, BoolValue(nil))

	bSlice := BoolSlice([]bool{false})
	bSliceVal := BoolSliceValue(bSlice)
	utils.AssertEqual(t, []bool{false}, bSliceVal)
	utils.AssertNil(t, BoolSlice(nil))
	utils.AssertNil(t, BoolSliceValue(nil))

	f64 := Float64(2.00)
	f64Val := Float64Value(f64)
	utils.AssertEqual(t, float64(2.00), f64Val)
	utils.AssertEqual(t, float64(0), Float64Value(nil))

	f32 := Float32(2.00)
	f32Val := Float32Value(f32)
	utils.AssertEqual(t, float32(2.00), f32Val)
	utils.AssertEqual(t, float32(0), Float32Value(nil))

	f64Slice := Float64Slice([]float64{2.00})
	f64SliceVal := Float64ValueSlice(f64Slice)
	utils.AssertEqual(t, []float64{2.00}, f64SliceVal)
	utils.AssertNil(t, Float64Slice(nil))
	utils.AssertNil(t, Float64ValueSlice(nil))

	f32Slice := Float32Slice([]float32{2.00})
	f32SliceVal := Float32ValueSlice(f32Slice)
	utils.AssertEqual(t, []float32{2.00}, f32SliceVal)
	utils.AssertNil(t, Float32Slice(nil))
	utils.AssertNil(t, Float32ValueSlice(nil))

	i := Int(1)
	iVal := IntValue(i)
	utils.AssertEqual(t, 1, iVal)
	utils.AssertEqual(t, 0, IntValue(nil))

	i8 := Int8(int8(1))
	i8Val := Int8Value(i8)
	utils.AssertEqual(t, int8(1), i8Val)
	utils.AssertEqual(t, int8(0), Int8Value(nil))

	i16 := Int16(int16(1))
	i16Val := Int16Value(i16)
	utils.AssertEqual(t, int16(1), i16Val)
	utils.AssertEqual(t, int16(0), Int16Value(nil))

	i32 := Int32(int32(1))
	i32Val := Int32Value(i32)
	utils.AssertEqual(t, int32(1), i32Val)
	utils.AssertEqual(t, int32(0), Int32Value(nil))

	i64 := Int64(int64(1))
	i64Val := Int64Value(i64)
	utils.AssertEqual(t, int64(1), i64Val)
	utils.AssertEqual(t, int64(0), Int64Value(nil))

	iSlice := IntSlice([]int{1})
	iSliceVal := IntValueSlice(iSlice)
	utils.AssertEqual(t, []int{1}, iSliceVal)
	utils.AssertNil(t, IntSlice(nil))
	utils.AssertNil(t, IntValueSlice(nil))

	i8Slice := Int8Slice([]int8{1})
	i8ValSlice := Int8ValueSlice(i8Slice)
	utils.AssertEqual(t, []int8{1}, i8ValSlice)
	utils.AssertNil(t, Int8Slice(nil))
	utils.AssertNil(t, Int8ValueSlice(nil))

	i16Slice := Int16Slice([]int16{1})
	i16ValSlice := Int16ValueSlice(i16Slice)
	utils.AssertEqual(t, []int16{1}, i16ValSlice)
	utils.AssertNil(t, Int16Slice(nil))
	utils.AssertNil(t, Int16ValueSlice(nil))

	i32Slice := Int32Slice([]int32{1})
	i32ValSlice := Int32ValueSlice(i32Slice)
	utils.AssertEqual(t, []int32{1}, i32ValSlice)
	utils.AssertNil(t, Int32Slice(nil))
	utils.AssertNil(t, Int32ValueSlice(nil))

	i64Slice := Int64Slice([]int64{1})
	i64ValSlice := Int64ValueSlice(i64Slice)
	utils.AssertEqual(t, []int64{1}, i64ValSlice)
	utils.AssertNil(t, Int64Slice(nil))
	utils.AssertNil(t, Int64ValueSlice(nil))

	ui := Uint(1)
	uiVal := UintValue(ui)
	utils.AssertEqual(t, uint(1), uiVal)
	utils.AssertEqual(t, uint(0), UintValue(nil))

	ui8 := Uint8(uint8(1))
	ui8Val := Uint8Value(ui8)
	utils.AssertEqual(t, uint8(1), ui8Val)
	utils.AssertEqual(t, uint8(0), Uint8Value(nil))

	ui16 := Uint16(uint16(1))
	ui16Val := Uint16Value(ui16)
	utils.AssertEqual(t, uint16(1), ui16Val)
	utils.AssertEqual(t, uint16(0), Uint16Value(nil))

	ui32 := Uint32(uint32(1))
	ui32Val := Uint32Value(ui32)
	utils.AssertEqual(t, uint32(1), ui32Val)
	utils.AssertEqual(t, uint32(0), Uint32Value(nil))

	ui64 := Uint64(uint64(1))
	ui64Val := Uint64Value(ui64)
	utils.AssertEqual(t, uint64(1), ui64Val)
	utils.AssertEqual(t, uint64(0), Uint64Value(nil))

	uiSlice := UintSlice([]uint{1})
	uiValSlice := UintValueSlice(uiSlice)
	utils.AssertEqual(t, []uint{1}, uiValSlice)
	utils.AssertNil(t, UintSlice(nil))
	utils.AssertNil(t, UintValueSlice(nil))

	ui8Slice := Uint8Slice([]uint8{1})
	ui8ValSlice := Uint8ValueSlice(ui8Slice)
	utils.AssertEqual(t, []uint8{1}, ui8ValSlice)
	utils.AssertNil(t, Uint8Slice(nil))
	utils.AssertNil(t, Uint8ValueSlice(nil))

	ui16Slice := Uint16Slice([]uint16{1})
	ui16ValSlice := Uint16ValueSlice(ui16Slice)
	utils.AssertEqual(t, []uint16{1}, ui16ValSlice)
	utils.AssertNil(t, Uint16Slice(nil))
	utils.AssertNil(t, Uint16ValueSlice(nil))

	ui32Slice := Uint32Slice([]uint32{1})
	ui32ValSlice := Uint32ValueSlice(ui32Slice)
	utils.AssertEqual(t, []uint32{1}, ui32ValSlice)
	utils.AssertNil(t, Uint32Slice(nil))
	utils.AssertNil(t, Uint32ValueSlice(nil))

	ui64Slice := Uint64Slice([]uint64{1})
	ui64ValSlice := Uint64ValueSlice(ui64Slice)
	utils.AssertEqual(t, []uint64{1}, ui64ValSlice)
	utils.AssertNil(t, Uint64Slice(nil))
	utils.AssertNil(t, Uint64ValueSlice(nil))
}

func Test_TransInterfaceToInt(t *testing.T) {
	a := TransInterfaceToInt(nil)
	utils.AssertNil(t, a)

	a = TransInterfaceToInt(10)
	utils.AssertEqual(t, IntValue(a), 10)
}

func Test_TransInterfaceToInt64(t *testing.T) {
	a := TransInterfaceToInt64(nil)
	utils.AssertNil(t, a)

	a = TransInterfaceToInt64(int64(10))
	utils.AssertEqual(t, Int64Value(a), int64(10))
}

func Test_TransInterfaceToString(t *testing.T) {
	a := TransInterfaceToString(nil)
	utils.AssertNil(t, a)

	a = TransInterfaceToString("10")
	utils.AssertEqual(t, StringValue(a), "10")
}

func Test_TransInt32AndInt(t *testing.T) {
	a := ToInt(Int32(10))
	utils.AssertEqual(t, IntValue(a), 10)

	b := ToInt32(a)
	utils.AssertEqual(t, Int32Value(b), int32(10))
}

func Test_ToString(t *testing.T) {
	str := ToString(10)
	utils.AssertEqual(t, "10", str)

	str = ToString("10")
	utils.AssertEqual(t, "10", str)
}

func Test_ToObject(t *testing.T) {
	str := "{sdsfdsd:"
	result := ToObject(str)
	utils.AssertNil(t, result)

	input := map[string]string{
		"name": "test",
	}
	result = ToObject(input)
	utils.AssertEqual(t, "test", result["name"].(string))
}

func Test_ToReader(t *testing.T) {
	str := "abc"
	reader := ToReader(String(str))
	byt, err := ioutil.ReadAll(reader)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "abc", string(byt))

	read := strings.NewReader("bcd")
	reader = ToReader(read)
	byt, err = ioutil.ReadAll(reader)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "bcd", string(byt))

	byts := []byte("cdf")
	reader = ToReader(byts)
	byt, err = ioutil.ReadAll(reader)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "cdf", string(byt))

	num := 10
	defer func() {
		err := recover()
		utils.AssertEqual(t, "Invalid Body. Please set a valid Body.", err.(string))
	}()
	reader = ToReader(num)
	byt, err = ioutil.ReadAll(reader)
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "", string(byt))
}
