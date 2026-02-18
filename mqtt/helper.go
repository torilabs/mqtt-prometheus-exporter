package mqtt

import "strings"

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
