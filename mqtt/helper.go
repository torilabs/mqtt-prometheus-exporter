package mqtt

import (
	"fmt"
	"strconv"
	"strings"
)

func getTopicPart(topic string, idx int) string {
	s := strings.Split(topic, "/")
	if idx > 0 && idx < len(s) {
		return s[idx]
	}
	if idx < 0 && idx >= -len(s) {
		return s[len(s)+idx]
	}
	return ""
}

func findInJSON(jsonMap map[string]interface{}, path string) (interface{}, bool) {
	if path == "" || len(jsonMap) == 0 {
		return nil, false
	}
	pp := strings.SplitN(path, ".", 2)
	if val, found := jsonMap[pp[0]]; found && len(pp) > 1 {
		if subJSONMap, ok := val.(map[string]interface{}); ok {
			return findInJSON(subJSONMap, pp[1])
		}
		return nil, false
	} else if found {
		return val, true
	}
	return nil, false
}

// labelValueOf converts a scalar JSON value to a label value.
// Null, arrays and objects are not supported.
func labelValueOf(v interface{}) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(t), true
	default:
		return "", false
	}
}

// booleanStrings are the case insensitive string forms converted to the value of a boolean.
var booleanStrings = map[string]float64{
	"true": 1, "t": 1, "yes": 1, "on": 1,
	"false": 0, "f": 0, "no": 0, "off": 0,
}

// jsonValueToFloat converts a JSON value to the value of a metric.
// Numbers and numeric strings are converted first. When that is not possible, booleans (true/false)
// and the strings true/false, t/f, yes/no and on/off (in any case) are converted to 1 or 0.
// The returned flag tells that the value was converted as a boolean.
func jsonValueToFloat(value interface{}) (float64, bool, error) {
	if b, ok := value.(bool); ok {
		if b {
			return 1, true, nil
		}
		return 0, true, nil
	}
	floatValue, err := strconv.ParseFloat(fmt.Sprintf("%v", value), 64)
	if err == nil {
		return floatValue, false, nil
	}
	if str, ok := value.(string); ok {
		if boolValue, found := booleanStrings[strings.ToLower(strings.TrimSpace(str))]; found {
			return boolValue, true, nil
		}
	}
	return 0, false, err
}
