package client

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/codenotary/immudb/pkg/api/schema"
	"github.com/codenotary/immudb/pkg/server"
	"github.com/codenotary/immudb/pkg/server/servertest"
	"github.com/codenotary/immudb/pkg/stream"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestImmuClient_StreamZScan(t *testing.T) {
	options := server.DefaultOptions().WithAuth(true)
	bs := servertest.NewBufconnServer(options)

	defer os.RemoveAll(options.Dir)
	defer os.Remove(".state-")

	bs.Start()
	defer bs.Stop()

	client, err := NewImmuClient(DefaultOptions().WithDialOptions(
		&[]grpc.DialOption{grpc.WithContextDialer(bs.Dialer), grpc.WithInsecure()},
	))
	require.NoError(t, err)
	lr, err := client.Login(context.TODO(), []byte(`immudb`), []byte(`immudb`))
	require.NoError(t, err)

	md := metadata.Pairs("authorization", lr.Token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	kvs := []*stream.KeyValue{}

	for i := 1; i <= 100; i++ {
		k := []byte(fmt.Sprintf("key-%d", i))
		v := []byte(fmt.Sprintf("val-%d", i))
		kv := &stream.KeyValue{
			Key: &stream.ValueSize{
				Content: bufio.NewReader(bytes.NewBuffer(k)),
				Size:    len(k),
			},
			Value: &stream.ValueSize{
				Content: bufio.NewReader(bytes.NewBuffer(v)),
				Size:    len(v),
			},
		}
		kvs = append(kvs, kv)
	}

	meta, err := client.StreamSet(ctx, kvs)
	require.NoError(t, err)
	require.NotNil(t, meta)

	set := "StreamZScanTestSet"
	setBytes := []byte(set)
	for i := range kvs {
		require.NoError(t, err)
		_, err = client.ZAdd(
			ctx, setBytes, float64((i+1)*10), []byte(fmt.Sprintf("key-%d", i+1)))
		require.NoError(t, err)
	}

	zScanResp, err := client.StreamZScan(ctx, &schema.ZScanRequest{Set: setBytes, SinceTx: meta.Id})

	client.Disconnect()

	require.Len(t, zScanResp.Entries, 100)
}

func TestImmuClient_StreamHistory(t *testing.T) {
	options := server.DefaultOptions().WithAuth(true)
	bs := servertest.NewBufconnServer(options)

	defer os.RemoveAll(options.Dir)
	defer os.Remove(".state-")

	bs.Start()
	defer bs.Stop()

	client, err := NewImmuClient(DefaultOptions().WithDialOptions(
		&[]grpc.DialOption{grpc.WithContextDialer(bs.Dialer), grpc.WithInsecure()},
	))
	require.NoError(t, err)
	lr, err := client.Login(context.TODO(), []byte(`immudb`), []byte(`immudb`))
	require.NoError(t, err)

	md := metadata.Pairs("authorization", lr.Token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	var meta *schema.TxMetadata

	k := []byte("StreamHistoryTestKey")
	for i := 1; i <= 100; i++ {
		v := []byte(fmt.Sprintf("val-%d", i))
		kv := &stream.KeyValue{
			Key: &stream.ValueSize{
				Content: bufio.NewReader(bytes.NewBuffer(k)),
				Size:    len(k),
			},
			Value: &stream.ValueSize{
				Content: bufio.NewReader(bytes.NewBuffer(v)),
				Size:    len(v),
			},
		}
		meta, err = client.StreamSet(ctx, []*stream.KeyValue{kv})
		require.NoError(t, err)
		require.NotNil(t, meta)
	}

	historyResp, err := client.StreamHistory(ctx, &schema.HistoryRequest{Key: k, SinceTx: meta.Id})

	client.Disconnect()

	require.Len(t, historyResp.Entries, 100)
}
