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
		name   string
		args   args
		want   interface{}
		wantOk bool
	}{
		{
			name: "1st level value",
			args: args{
				path: "city",
			},
			want:   "Tokyo",
			wantOk: true,
		},
		{
			name: "2nd level value",
			args: args{
				path: "temperatures.in",
			},
			want:   22.15,
			wantOk: true,
		},
		{
			name: "object value",
			args: args{
				path: "temperatures",
			},
			want:   map[string]interface{}{"out": 12.5, "in": 22.15},
			wantOk: true,
		},
		{
			name: "value not found on 1st level",
			args: args{
				path: "notdefined",
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "value not found on 2nd level",
			args: args{
				path: "temperatures.notdefined",
			},
			want:   nil,
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr := []byte(`{"city":"Tokyo", "temperatures": {"out": 12.5, "in": 22.15}, "size": -5}`)
			jsonMap := make(map[string]interface{})
			if err := json.Unmarshal(jsonStr, &jsonMap); err != nil {
				t.Fatal(err)
			}
			if got, gotOk := findInJSON(jsonMap, tt.args.path); !reflect.DeepEqual(got, tt.want) || gotOk != tt.wantOk {
				t.Errorf("findInJSON() = (%v, %v), want (%v, %v)", got, gotOk, tt.want, tt.wantOk)
			}
		})
	}
}

func Test_jsonValueToFloat(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		want        float64
		wantBoolean bool
		wantErr     bool
	}{
		{name: "number", value: 12.5, want: 12.5},
		{name: "negative number", value: -5.0, want: -5},
		{name: "numeric string", value: "12.5", want: 12.5},
		{name: "numeric string 1 stays numeric", value: "1", want: 1},
		{name: "numeric string 0 stays numeric", value: "0", want: 0},
		{name: "number 2 stays numeric", value: 2.0, want: 2},
		{name: "boolean true", value: true, want: 1, wantBoolean: true},
		{name: "boolean false", value: false, want: 0, wantBoolean: true},
		{name: "string true", value: "true", want: 1, wantBoolean: true},
		{name: "string false", value: "false", want: 0, wantBoolean: true},
		{name: "string TRUE", value: "TRUE", want: 1, wantBoolean: true},
		{name: "string t", value: "t", want: 1, wantBoolean: true},
		{name: "string F", value: "F", want: 0, wantBoolean: true},
		{name: "string yes", value: "Yes", want: 1, wantBoolean: true},
		{name: "string no", value: "no", want: 0, wantBoolean: true},
		{name: "string on", value: "On", want: 1, wantBoolean: true},
		{name: "string off with spaces", value: "  OFF ", want: 0, wantBoolean: true},
		{name: "unknown string", value: "maybe", wantErr: true},
		{name: "empty string", value: "", wantErr: true},
		{name: "blank string", value: "  ", wantErr: true},
		{name: "text", value: "Tokyo", wantErr: true},
		{name: "null", value: nil, wantErr: true},
		{name: "object", value: map[string]interface{}{"a": 1.0}, wantErr: true},
		{name: "array", value: []interface{}{true}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isBoolean, err := jsonValueToFloat(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("jsonValueToFloat() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got != tt.want {
				t.Errorf("jsonValueToFloat() = %v, want %v", got, tt.want)
			}
			if isBoolean != tt.wantBoolean {
				t.Errorf("jsonValueToFloat() isBoolean = %v, want %v", isBoolean, tt.wantBoolean)
			}
		})
	}
}
