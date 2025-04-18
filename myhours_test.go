package myhours

import (
	"reflect"
	"testing"
)

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

func Test_byIndex(t *testing.T) {
	type args[T comparable] struct {
		set   []T
		index int
	}
	type testCase[T comparable] struct {
		name string
		args args[T]
		want T
	}
	tests := []testCase[int]{
		{
			name: "index out of bounds, negative",
			args: args[int]{
				set:   []int{1, 2, 3},
				index: -1,
			},
			want: 0,
		},
		{
			name: "index out of bounds, positive",
			args: args[int]{
				set:   []int{1, 2, 3},
				index: 4,
			},
			want: 0,
		},
		{
			name: "return correct value",
			args: args[int]{
				set:   []int{1, 2, 3},
				index: 1,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexOrZero(tt.args.set, tt.args.index); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("byIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_findCategory(t *testing.T) {
	type args struct {
		categories []Category
		id         int64
	}
	tests := []struct {
		name string
		args args
		want Category
	}{
		{
			name: "not found",
			args: args{
				categories: []Category{
					{ID: 4, Name: "4"},
					{ID: 6, Name: "6"},
				},
				id: 2,
			},
			want: Category{ID: 0, Name: "unknown"},
		},
		{
			name: "found",
			args: args{
				categories: []Category{
					{ID: 4, Name: "4"},
					{ID: 6, Name: "6"},
				},
				id: 6,
			},
			want: Category{ID: 6, Name: "6"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findCategory(tt.args.categories, tt.args.id); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findCategory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nextCategoryID(t *testing.T) {
	type args struct {
		categories []Category
		currentID  int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "return next ID",
			args: args{
				categories: []Category{
					{ID: 4, Name: "4"},
					{ID: 5, Name: "5"},
					{ID: 6, Name: "6"},
				},
				currentID: 5,
			},
			want: 6,
		},
		{
			name: "current is last",
			args: args{
				categories: []Category{
					{ID: 4, Name: "4"},
					{ID: 5, Name: "5"},
					{ID: 6, Name: "6"},
				},
				currentID: 6,
			},
			want: 4,
		},
		{
			name: "current is first",
			args: args{
				categories: []Category{
					{ID: 4, Name: "4"},
					{ID: 5, Name: "5"},
					{ID: 6, Name: "6"},
				},
				currentID: 4,
			},
			want: 5,
		},
		{
			name: "current is not found",
			args: args{
				categories: []Category{
					{ID: 2, Name: "2"},
					{ID: 7, Name: "7"},
				},
				currentID: 6,
			},
			want: 2,
		},
		{
			name: "slice is empty",
			args: args{
				categories: []Category{},
				currentID:  6,
			},
			want: 6,
		},
		{
			name: "slice is nil",
			args: args{
				categories: nil,
				currentID:  6,
			},
			want: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextCategoryID(tt.args.categories, tt.args.currentID); got != tt.want {
				t.Errorf("nextCategoryID() = %v, want %v", got, tt.want)
			}
		})
	}
}
