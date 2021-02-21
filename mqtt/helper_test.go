package mqtt

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_getTopicPart(t *testing.T) {
	type args struct {
		topic string
		idx   int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Zero index",
			args: args{
				topic: "/level1/level2/level3",
				idx:   0,
			},
			want: "",
		},
		{
			name: "Positive index",
			args: args{
				topic: "/level1/level2",
				idx:   2,
			},
			want: "level2",
		},
		{
			name: "Negative index",
			args: args{
				topic: "/level1/level2/level3",
				idx:   -2,
			},
			want: "level2",
		},
		{
			name: "Positive index out of range",
			args: args{
				topic: "/level1/level2",
				idx:   3,
			},
			want: "",
		},
		{
			name: "Negative index out of range",
			args: args{
				topic: "/level1/level2",
				idx:   -3,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTopicPart(tt.args.topic, tt.args.idx); got != tt.want {
				t.Errorf("getTopicPart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_findInJson(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			name: "1st level value",
			args: args{
				path: "city",
			},
			want: "Tokyo",
		},
		{
			name: "2nd level value",
			args: args{
				path: "temperatures.in",
			},
			want: 22.15,
		},
		{
			name: "object value",
			args: args{
				path: "temperatures",
			},
			want: map[string]interface{}{"out": 12.5, "in": 22.15},
		},
		{
			name: "value not found on 1st level",
			args: args{
				path: "notdefined",
			},
			want: nil,
		},
		{
			name: "value not found on 2nd level",
			args: args{
				path: "temperatures.notdefined",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr := []byte(`{"city":"Tokyo", "temperatures": {"out": 12.5, "in": 22.15}, "size": -5}`)
			jsonMap := make(map[string]interface{})
			if err := json.Unmarshal(jsonStr, &jsonMap); err != nil {
				t.Fatal(err)
			}
			if got := findInJSON(jsonMap, tt.args.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findInJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
