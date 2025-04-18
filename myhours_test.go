package myhours

import "testing"

func Test_incMax(t *testing.T) {
	type args struct {
		v   int
		max int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "two below max", args: args{v: 0, max: 2}, want: 1},
		{name: "from one below max", args: args{v: 1, max: 2}, want: 2},
		{name: "from max", args: args{v: 2, max: 2}, want: 2},
		{name: "from above max", args: args{v: 3, max: 2}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := incMax(tt.args.v, tt.args.max); got != tt.want {
				t.Errorf("incMax() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decMax(t *testing.T) {
	type args struct {
		v   int
		max int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "two below max", args: args{v: 0, max: 2}, want: -1},
		{name: "from one below max", args: args{v: 1, max: 2}, want: 0},
		{name: "from max", args: args{v: 2, max: 2}, want: 1},
		{name: "from above max", args: args{v: 3, max: 2}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decMax(tt.args.v, tt.args.max); got != tt.want {
				t.Errorf("decMax() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_incWrap(t *testing.T) {
	type args struct {
		v   int
		min int
		max int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "two below max", args: args{v: 0, min: 0, max: 2}, want: 1},
		{name: "from one below max", args: args{v: 1, min: 0, max: 2}, want: 2},
		{name: "from max", args: args{v: 2, min: 0, max: 2}, want: 0},
		{name: "from above max", args: args{v: 3, min: 0, max: 2}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := incWrap(tt.args.v, tt.args.min, tt.args.max); got != tt.want {
				t.Errorf("incWrap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decWrap(t *testing.T) {
	type args struct {
		v   int
		min int
		max int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "two above min", args: args{v: 2, min: 0, max: 2}, want: 1},
		{name: "from one above min", args: args{v: 1, min: 0, max: 2}, want: 0},
		{name: "from min", args: args{v: 0, min: 0, max: 2}, want: 2},
		{name: "from below min", args: args{v: -1, min: 0, max: 2}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decWrap(tt.args.v, tt.args.min, tt.args.max); got != tt.want {
				t.Errorf("decWrap() = %v, want %v", got, tt.want)
			}
		})
	}
}
