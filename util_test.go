package neo4j_tracing

import "testing"

func Test_spanName(t *testing.T) {
	type args struct {
		operation string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple operation",
			args: args{operation: "Query"},
			want: "neo4j.Query",
		},
		{
			name: "empty operation",
			args: args{operation: ""},
			want: "neo4j.",
		},
		{
			name: "operation with spaces",
			args: args{operation: "Begin Transaction"},
			want: "neo4j.Begin Transaction",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := spanName(tt.args.operation); got != tt.want {
				t.Errorf("spanName() = %v, want %v", got, tt.want)
			}
		})
	}
}
