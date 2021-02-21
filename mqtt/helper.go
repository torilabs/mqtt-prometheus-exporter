package mqtt

import "strings"

func getTopicPart(topic string, idx int) string {
	s := strings.Split(topic, "/")
	switch true {
	case idx > 0 && idx < len(s):
		return s[idx]
	case idx < 0 && idx >= -len(s):
		return s[len(s)+idx]
	default:
		return ""
	}
}

func findInJSON(jsonMap map[string]interface{}, path string) interface{} {
	if path == "" || len(jsonMap) == 0 {
		return nil
	}
	pp := strings.SplitN(path, ".", 2)
	val := jsonMap[pp[0]]
	if val == nil || len(pp) == 1 {
		return val
	}
	subJSONMap, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}
	return findInJSON(subJSONMap, pp[1])
}
