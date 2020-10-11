package mqtt

import "testing"

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
