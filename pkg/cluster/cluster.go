package cluster

import "context"

type (
	Cluster interface {
		AddNode(ctx context.Context) (Node, error)
		Nodes(ctx context.Context) ([]Node, error)
		Delete(ctx context.Context) error
	}

	Node interface {
		Finish(ctx context.Context, result []byte) error
		Result(ctx context.Context) ([]byte, error)
		Delete(ctx context.Context) error
	}
)
