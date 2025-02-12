package dara

import (
	"encoding/json"
	"testing"

	"github.com/alibabacloud-go/tea/utils"
)

type validatorTest struct {
	Num  *int       `json:"num" require:"true"`
	Str  *string    `json:"str" pattern:"^[a-d]*$" maxLength:"4"`
	Test *errLength `json:"test"`
	List []*string  `json:"list" pattern:"^[a-d]*$" maxLength:"4"`
}

type GetBucketLocationResponse struct {
	RequestId          *string `json:"x-oss-request-id" xml:"x-oss-request-id" require:"true"`
	LocationConstraint *string `json:"LocationConstraint" xml:"LocationConstraint" require:"true"`
}

type errLength struct {
	Num *int `json:"num" maxLength:"a"`
}

func Test_ParseXml(t *testing.T) {
	str := `<?xml version="1.0" encoding="utf-8" standalone="no"?>
	<num>10</num>`
	result := ParseXml(str, new(validatorTest))
	utils.AssertEqual(t, 1, len(result))
	str = `<?xml version="1.0" encoding="utf-8" standalone="no"?>
	<num></num>`
	result = ParseXml(str, new(validatorTest))
	utils.AssertEqual(t, 1, len(result))
	xmlVal := `<?xml version="1.0" encoding="utf-8" standalone="no"?>
<students>
    <student number="1001">
        <name>zhangSan</name>
        <age>23</age>
        <sex>male</sex>
    </student>
</students>`
	res := ParseXml(xmlVal, nil)
	utils.AssertEqual(t, 1, len(res))
}

func Test_ToXML(t *testing.T) {
	val := map[string]interface{}{
		"oss": map[string]interface{}{
			"key": "value",
		},
	}
	str := ToXML(val)
	utils.AssertEqual(t, "<oss><key>value</key></oss>", str)
}

func Test_getStartElement(t *testing.T) {
	xmlVal := `<?xml version="1.0" encoding="utf-8" standalone="no"?>
<students>
    <student number="1001">
        <name>zhangSan</name>
        <age>23</age>
        <sex>male</sex>
    </student>
</students>`
	str := getStartElement([]byte(xmlVal))
	utils.AssertEqual(t, "students", str)

	xmlVal = `<?xml version="1.0" encoding="utf-8" standalone="no"?>
<students/\>
    <student number="1001">
        <name>zhangSan</name>
        <age>23</age>
        <sex>male</sex>
    </student>
</students>`
	str = getStartElement([]byte(xmlVal))
	utils.AssertEqual(t, "", str)
}

func Test_mapToXML(t *testing.T) {
	obj := map[string]interface{}{
		"struct": map[string]interface{}{
			"param1": "value1",
			"list":   []string{"value2", "value3"},
			"listMap": []map[string]interface{}{
				map[string]interface{}{
					"param2": "value2",
				},
				map[string]interface{}{
					"param3": "value3",
				},
			},
			"listMapString": []map[string]string{
				map[string]string{
					"param4": "value4",
				},
				map[string]string{
					"param5": "value5",
				},
			},
			"mapString": map[string]string{
				"param6": "value6",
			},
			"listInterface": []interface{}{"10", 20},
		},
	}
	byt, _ := json.Marshal(obj)
	obj1 := make(map[string]interface{})
	json.Unmarshal(byt, &obj1)
	xml := mapToXML(obj1)
	utils.AssertContains(t, xml, `<listInterface>10</listInterface>`)
}

func Test_XmlUnmarshal(t *testing.T) {
	result := new(GetBucketLocationResponse)
	xmlVal := `<?xml version="1.0" encoding="UTF-8"?>
<LocationConstraint>oss-cn-hangzhou</LocationConstraint >`
	out, err := xmlUnmarshal([]byte(xmlVal), result)
	utils.AssertNil(t, err)

	byt, _ := json.Marshal(out)
	utils.AssertEqual(t, `"oss-cn-hangzhou"`, string(byt))
}
