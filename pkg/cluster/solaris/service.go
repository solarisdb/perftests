package solaris

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/solarisdb/perftests/pkg/cluster"
	"github.com/solarisdb/solaris/api/gen/solaris/v1"
	"github.com/solarisdb/solaris/golibs/logging"
	"github.com/solarisdb/solaris/golibs/ulidutils"
)

type (
	solarCluster struct {
		clusterID    string
		clusterLogID string
		solaris      solaris.ServiceClient
		logger       logging.Logger
	}

	solarisNode struct {
		nodeID    string
		nodeLogID string
		cluster   *solarCluster
	}

	clusterRecord struct {
		NodeID    string `json:"node_id"`
		NodeLogID string `json:"node_log_id"`
	}

	nodeResult struct {
		Result string `json:"result"`
	}
)

const prefix = "solarisdb.perftests.cluster"

var _ cluster.Cluster = (*solarCluster)(nil)
var _ cluster.Node = (*solarisNode)(nil)

func NewCluster(ctx context.Context, clusterID string, solaris solaris.ServiceClient) (cluster.Cluster, error) {
	sc := new(solarCluster)
	sc.clusterID = clusterID
	sc.solaris = solaris
	sc.logger = logging.NewLogger("solarisCluster")
	clusterLogID, err := sc.getOrCreateLog(ctx)
	sc.clusterLogID = clusterLogID
	return sc, err
}

func (s *solarCluster) AddNode(ctx context.Context) (cluster.Node, error) {
	nodeID := ulidutils.NewUUID().String()
	node, err := newNode(ctx, nodeID, s)
	if err != nil {
		return nil, err
	}
	if err := s.addNode(ctx, s.clusterLogID, node); err != nil {
		return nil, err
	}
	return node, nil
}

func (s *solarCluster) Nodes(ctx context.Context) ([]cluster.Node, error) {
	var nodes []cluster.Node
	fromID := ""
	for {
		req := &solaris.QueryRecordsRequest{
			LogIDs:        []string{s.clusterLogID},
			Limit:         100,
			StartRecordID: fromID,
		}
		res, err := s.solaris.QueryRecords(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to query nodes: %w", err)
		}
		for _, nodeRec := range res.Records {
			var node clusterRecord
			_ = json.Unmarshal(nodeRec.Payload, &node)
			nodes = append(nodes, &solarisNode{
				cluster:   s,
				nodeID:    node.NodeID,
				nodeLogID: node.NodeLogID,
			})
		}
		fromID = res.NextPageID
		if fromID == "" {
			break
		}
	}
	return nodes, nil
}

func (s *solarCluster) getOrCreateLog(ctx context.Context) (string, error) {
	qRes, err := s.solaris.QueryLogs(ctx, &solaris.QueryLogsRequest{
		Condition: fmt.Sprintf("tag(%q)=%q", prefix, s.clusterID),
		Limit:     1,
	})
	if err != nil {
		return "", err
	}
	if len(qRes.Logs) == 0 {
		s.logger.Tracef("cluster log not found, going to create it")
		log, err := s.solaris.CreateLog(ctx, &solaris.Log{
			Tags: map[string]string{
				prefix: s.clusterID,
			},
		})
		if err != nil {
			return "", err
		}
		s.logger.Tracef("cluster log created %s", log.ID)
		return log.ID, nil
	}
	s.logger.Tracef("found active cluster log %s", qRes.Logs[0].ID)
	return qRes.Logs[0].ID, nil
}

func (s *solarCluster) addNode(ctx context.Context, clusterLogID string, node *solarisNode) error {
	nodeRec, _ := json.Marshal(clusterRecord{NodeID: node.nodeID, NodeLogID: node.nodeLogID})
	_, err := s.solaris.AppendRecords(ctx, &solaris.AppendRecordsRequest{
		LogID: clusterLogID,
		Records: []*solaris.Record{
			{Payload: nodeRec},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to append node to cluster log %s: %w", clusterLogID, err)
	}
	return nil
}

func (s *solarCluster) Delete(ctx context.Context) error {
	nodes, err := s.Nodes(ctx)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		_ = node.Delete(ctx)
	}
	_, err = s.solaris.DeleteLogs(ctx, &solaris.DeleteLogsRequest{
		Condition: fmt.Sprintf("logID='%s'", s.clusterLogID),
	})
	return err
}

func (s *solarCluster) String() string {
	return fmt.Sprintf("Cluster{ID=%s}", s.clusterID)
}

func newNode(ctx context.Context, nodeID string, cluster *solarCluster) (*solarisNode, error) {
	sc := new(solarisNode)
	sc.nodeID = nodeID
	sc.cluster = cluster
	nodeLogID, err := sc.getOrCreateLog(ctx)
	sc.nodeLogID = nodeLogID
	return sc, err
}

func (s *solarisNode) Finish(ctx context.Context, result []byte) error {
	//nodeRec, _ := json.Marshal(nodeResult{Result: string(result)})
	_, err := s.cluster.solaris.AppendRecords(ctx, &solaris.AppendRecordsRequest{
		LogID: s.nodeLogID,
		Records: []*solaris.Record{
			{Payload: result},
		},
	})
	return err
}

func (s *solarisNode) Delete(ctx context.Context) error {
	_, err := s.cluster.solaris.DeleteLogs(ctx, &solaris.DeleteLogsRequest{
		Condition: fmt.Sprintf("logID='%s'", s.nodeLogID),
	})
	return err
}

func (s *solarisNode) Result(ctx context.Context) ([]byte, error) {
	//var node nodeResult
	fromID := ""
	for {
		req := &solaris.QueryRecordsRequest{
			LogIDs:        []string{s.nodeLogID},
			Limit:         1,
			StartRecordID: fromID,
		}
		res, err := s.cluster.solaris.QueryRecords(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to query node result: %w", err)
		}
		if len(res.Records) == 0 {
			time.Sleep(5 * time.Second)
			continue
		}
		rec := res.Records[0]
		//_ = json.Unmarshal(rec.Payload, &node)
		return rec.Payload, nil
	}
	//return cluster.Result(node.Result), nil
}

func (s *solarisNode) getOrCreateLog(ctx context.Context) (string, error) {
	qRes, err := s.cluster.solaris.QueryLogs(ctx, &solaris.QueryLogsRequest{
		Condition: fmt.Sprintf("tag('%s')='%s'", prefix, s.nodeID),
		Limit:     1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to query node log %w", err)
	}
	if len(qRes.Logs) == 0 {
		log, err := s.cluster.solaris.CreateLog(ctx, &solaris.Log{
			Tags: map[string]string{
				prefix: s.nodeID,
			},
		})
		if err != nil {
			return "", fmt.Errorf("failed to create new node log %w", err)
		}
		return log.ID, nil
	}
	return qRes.Logs[0].ID, nil
}

func (s *solarisNode) String() string {
	return fmt.Sprintf("Node{ID=%s %s}", s.nodeID, s.cluster)
}
