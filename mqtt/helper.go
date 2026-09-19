package mqtt

import (
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
