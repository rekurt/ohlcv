package market

import "testing"

func Test_flipMarkets(t *testing.T) {
	type args struct {
		s   string
		sep string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "",
			args: args{
				s:   "BTC/PXP",
				sep: "/",
			},
			want: "PXP_BTC",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := flipMarkets(tt.args.s, tt.args.sep); got != tt.want {
				t.Errorf("flipMarkets() = %v, want %v", got, tt.want)
			}
		})
	}
}
