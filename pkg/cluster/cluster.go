package cluster

import "context"

type (
	Cluster interface {
		AddNode(ctx context.Context) (Node, error)
		Nodes(ctx context.Context) ([]Node, error)
		Delete(ctx context.Context) error
	}

	Result string

	Node interface {
		Finish(ctx context.Context, result Result) error
		Result(ctx context.Context) (Result, error)
		Delete(ctx context.Context) error
	}
)
