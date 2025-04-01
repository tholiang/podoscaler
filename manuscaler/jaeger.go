package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"google.golang.org/grpc"
)

type JaegerGRPCClient struct {
	conn   *grpc.ClientConn
	client api_v2.QueryServiceClient
}

func NewJaegerGRPCClient(addr string) (*JaegerGRPCClient, error) {
	conn, err := grpc.NewClient(addr)
	if err != nil {
		return nil, err
	}
	return &JaegerGRPCClient{
		conn:   conn,
		client: api_v2.NewQueryServiceClient(conn),
	}, nil
}

func (jc *JaegerGRPCClient) GetLatencyMetrics(ctx context.Context, serviceName string, lookback time.Duration) (map[string]float64, error) {
	end := time.Now()
	start := end.Add(-lookback)

	resp, err := jc.client.FindTraces(ctx, &api_v2.FindTracesRequest{
		Query: &api_v2.TraceQueryParameters{
			ServiceName:  serviceName,
			StartTimeMin: start,
			StartTimeMax: end,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query traces: %w", err)
	}

	metrics := make(map[string]float64)

	for {
		traceResponse, err := resp.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to receive trace: %w", err)
		}

		// TODO: calculate something
		for _, span := range traceResponse.Spans {
		}
	}

	return metrics, nil
}
