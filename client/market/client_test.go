package market

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_List_manual(t *testing.T) {
	t.Skip()
	cli, err := New(
		Config{ServerURL: "https://master.api.stage.exchange.pointpay.io", ServerTLS: true},
		&ErrorProcessor{},
		map[interface{}]Option{},
		"",
	)
	require.NoError(t, err)
	list, err := cli.List(context.Background())
	require.NoError(t, err)
	for _, m := range list {
		fmt.Printf("%+v\n", m)
	}
}
