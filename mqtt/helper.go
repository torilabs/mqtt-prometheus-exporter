package mqtt

import "strings"

// getTopicPart returns the idx-th "/"-separated segment of the topic without allocating.
// Positive indexes count from the start (0 is unsupported and yields ""), negative from the end.
func getTopicPart(topic string, idx int) string {
	n := strings.Count(topic, "/") + 1
	target := idx
	switch {
	case idx > 0 && idx < n:
	case idx < 0 && idx >= -n:
		target = n + idx
	default:
		return ""
	}
	i := 0
	for part := range strings.SplitSeq(topic, "/") {
		if i == target {
			return part
		}
		i++
	}
	return ""
}

func findInJSON(jsonMap map[string]any, path string) (any, bool) {
	if path == "" || len(jsonMap) == 0 {
		return nil, false
	}
	pp := strings.SplitN(path, ".", 2)
	if val, found := jsonMap[pp[0]]; found && len(pp) > 1 {
		if subJSONMap, ok := val.(map[string]any); ok {
			return findInJSON(subJSONMap, pp[1])
		}
		return nil, false
	} else if found {
		return val, true
	}
	return nil, false
}
