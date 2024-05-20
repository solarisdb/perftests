package server

import (
	"fmt"
	"github.com/solarisdb/perftests/pkg/utils"
	"math"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/perftests/pkg/runner/cluster"
	"github.com/solarisdb/perftests/pkg/runner/solaris"
)

var defaultAddress = "localhost:50051"
var defaultEnvVarAddress = "PERFTESTS_SOLARIS_ADDRESS"
var defaultEnvRunID = "PERFTESTS_RUN_ID"
var defLogLevel = "info"

var oneKb = int(math.Pow(float64(2), float64(10)))
var oneMB = oneKb * oneKb
var oneGB = oneKb * oneMB

const appendToMetricName = "AppendTimeout"
const appendMsgsPerSecMetricName = "AppendMsgsInSec"
const appendBytesPerSecMetricName = "AppendBytesInSec"
const queryToMetricName = "QueryTimeout"
const queryMsgsPerSecMetricName = "QueryMsgsInSec"
const queryBytesPerSecMetricName = "QueryBytesInSec"

type (
	OpType    string
	AppendCfg struct {
		// ConcurrentLogs - how many logs are written concurrently
		ConcurrentLogs int
		// LogSize - how many data should be written to one log
		LogSize int
		// WritersForOneLog - how many writers to one log work concurrently
		WritersForOneLog int
		// BatchSize - how many records are written by one Append call
		BatchSize int
		// MsgSize - message size in bytes
		MsgSize int
	}
	QueryCfg struct {
		// ConcurrentLogs - how many logs are read concurrently
		ConcurrentLogs int
		// LogSize - how many data should be read from one log
		LogSize int
		// ReadersFromOneLog - how many writers to one log work concurrently
		ReadersFromOneLog int
		// QueryStep - how many records are written by one Append call
		QueryStep int
		// MsgSize - message size in bytes
		MsgSize int
	}
)

const (
	Append   OpType = "append"
	Cleanup  OpType = "cleanup"
	Sleep    OpType = "sleep"
	SeqQuery OpType = "seq_query"
)

func BuildConfig(opType OpType, params any) *model.Config {
	switch opType {
	case Append:
		cfg, _ := params.(*AppendCfg)
		return buildAppendToManyLogsTests(cfg)
	case Cleanup:
		return &model.Config{
			Tests: []model.Test{*cleanupCluster()},
		}
	case Sleep:
		return &model.Config{
			Tests: []model.Test{*pause()},
		}
	case SeqQuery:
		cfg, _ := params.(*QueryCfg)
		return buildQueryLogsTests(cfg)
	}
	return &model.Config{}
}

func cleanupCluster() *model.Test {
	scenario := clusterCleanup(defaultEnvRunID, defaultAddress, defaultEnvVarAddress)
	return &model.Test{
		Name:     fmt.Sprintf("Cleanup cluster"),
		Scenario: *scenario,
	}
}

func pause() *model.Test {
	scenario := &model.Scenario{
		Name: runner.PauseRunName,
		Config: model.ToScenarioConfig(&runner.PauseCfg{
			Value: "10000h",
		}),
	}
	return &model.Test{
		Name:     fmt.Sprintf("Sleep..."),
		Scenario: *scenario,
	}
}

func buildQueryLogsTests(cfg *QueryCfg) *model.Config {
	return &model.Config{
		Tests: []model.Test{
			*fillAndSeqReadManyLogs(cfg.ConcurrentLogs, cfg.ReadersFromOneLog, cfg.LogSize, cfg.QueryStep, cfg.MsgSize),
		},
	}
}

func buildAppendToManyLogsTests(cfg *AppendCfg) *model.Config {
	return &model.Config{
		Log: model.LoggingConfig{Level: defLogLevel},
		Tests: []model.Test{
			*appendToManyLogs(cfg.ConcurrentLogs, cfg.WritersForOneLog, cfg.LogSize, cfg.BatchSize, cfg.MsgSize),
		},
	}
}

// fillAndSeqReadManyLogs fills then reads logs
func fillAndSeqReadManyLogs(concurrentLogs, readers int, logSize, queryStep, msgSize int) *model.Test {
	appendBatchSize := 50 * oneMB
	batchSize := appendBatchSize / msgSize
	appendsToLog := logSize / msgSize / batchSize
	readCount := logSize / queryStep / msgSize
	test := appendToLogsThenQueryTest(defaultEnvRunID, defaultAddress, defaultEnvVarAddress,
		concurrentLogs,
		1, appendsToLog, batchSize, msgSize,
		readers, queryStep, readCount)
	test.Name = fmt.Sprintf("Seq read %d logs (by %d readers from each one), read %s by each reader sequentially from earliest to lates, one query: %d messages by %s",
		concurrentLogs,
		readers,
		utils.HumanReadableBytes(float64(logSize)),
		queryStep,
		utils.HumanReadableBytes(float64(msgSize)))
	return test
}

// appendToManyLogs appends to concurrentLogs logs
func appendToManyLogs(concurrentLogs, writers int, logSize, batchSize, oneStep int) *model.Test {
	appendsToLog := logSize / writers / oneStep / batchSize
	test := appendToLogsThenQueryTest(defaultEnvRunID, defaultAddress, defaultEnvVarAddress, concurrentLogs, writers, appendsToLog, batchSize, oneStep, 0, 100, -1)
	test.Name = fmt.Sprintf("Append to %d logs (by %d writers to each one), write %s to each log, one append: %d messages by %s",
		concurrentLogs,
		writers,
		utils.HumanReadableBytes(float64(logSize)),
		batchSize,
		utils.HumanReadableBytes(float64(oneStep)))
	return test
}

// concurrentLogs - how many logs are written concurrently
// writersToLog - how many writers to one log work concurrently
// appendsToLog - how many Append() will be called to one log
// batchSize - how many records are written on one Append call
// msgSize - message size in bytes
// logReaders - how many readers to one log work concurrently
// queryStep -  how many records are read on one Query call
// queriesNumber - how many Query() will be called for one log
func appendToLogsThenQueryTest(runID, svcAddress, envVarAddress string, concurrentLogs, writersToLog, appendsToLog, batchSize, msgSize int,
	logReaders, queryStep, queriesNumber int) *model.Test {
	scenario := appendToLogsThenQueryScenario(svcAddress, envVarAddress, concurrentLogs, writersToLog, appendsToLog, batchSize, msgSize,
		logReaders, queryStep, queriesNumber)
	scenario = clusterRun(runID, svcAddress, envVarAddress, scenario)
	return &model.Test{
		Name:     fmt.Sprintf("Append to %d logs then read it", concurrentLogs),
		Scenario: *scenario,
	}
}

func clusterRun(runID, svcAddress, envVarAddress string, wrappedScenario *model.Scenario) *model.Scenario {
	return &model.Scenario{
		Name: runner.SequenceRunName,
		Config: model.ToScenarioConfig(&runner.SequenceCfg{
			Steps: []model.Scenario{
				// connect to cluster, add node
				{
					Name: cluster.ConnectName,
					Config: model.ToScenarioConfig(&cluster.ConnectCfg{
						Address:       svcAddress,
						EnvVarAddress: envVarAddress,
						EnvRunID:      runID,
					}),
				},
				*wrappedScenario,
				// finish and wait other cluster nodes
				{
					Name: cluster.FinishName,
					Config: model.ToScenarioConfig(&cluster.FinishCfg{
						Await: true,
						Metrics: map[runner.MetricsType][]string{
							runner.DURATION: {appendToMetricName, queryToMetricName},
							runner.RPS:      {appendMsgsPerSecMetricName, appendBytesPerSecMetricName, queryMsgsPerSecMetricName, queryBytesPerSecMetricName},
						},
					}),
				},
				// delete cluster
				{
					Name: cluster.DeleteClusterName,
				},
			},
		}),
	}
}

func clusterCleanup(runID, svcAddress, envVarAddress string) *model.Scenario {
	return &model.Scenario{
		Name: runner.SequenceRunName,
		Config: model.ToScenarioConfig(&runner.SequenceCfg{
			Steps: []model.Scenario{
				// connect to cluster, add node
				{
					Name: cluster.ConnectName,
					Config: model.ToScenarioConfig(&cluster.ConnectCfg{
						Address:       svcAddress,
						EnvVarAddress: envVarAddress,
						EnvRunID:      runID,
					}),
				},
				// delete cluster
				{
					Name: cluster.DeleteClusterName,
				},
			},
		}),
	}
}

func appendToLogsThenQueryScenario(svcAddress, envVarAddress string, concurrentLogs, writersToLog, appendsToLog, batchSize, msgSize int,
	logReaders, queryStep, queriesNumber int) *model.Scenario {
	appendMetricName := appendToMetricName
	queryMetricName := queryToMetricName
	appendMsgsMName := appendMsgsPerSecMetricName
	appendBytesMName := appendBytesPerSecMetricName
	queryMsgsMName := queryMsgsPerSecMetricName
	queryBytesMName := queryBytesPerSecMetricName
	return &model.Scenario{
		Name: runner.SequenceRunName,
		Config: model.ToScenarioConfig(&runner.SequenceCfg{
			Steps: []model.Scenario{
				// connect to solaris
				{
					Name: solaris.ConnectName,
					Config: model.ToScenarioConfig(&solaris.ConnectCfg{
						Address:       svcAddress,
						EnvVarAddress: envVarAddress,
					}),
				},
				// init metrics
				{
					Name: runner.MetricsCreateRunName,
					Config: model.ToScenarioConfig(&runner.MetricsCreateCfg{
						Metrics: map[runner.MetricsType][]string{
							runner.DURATION: {appendMetricName, queryMetricName},
							runner.RPS:      {appendMsgsMName, appendBytesMName, queryMsgsMName, queryBytesMName},
						},
					}),
				},
				// append to 'concurrentLogs' number of logs in parallel, each appender adds 100 messages in parallel
				{
					Name: runner.RepeatRunName,
					Config: model.ToScenarioConfig(&runner.RepeatCfg{
						Count:    concurrentLogs,
						Executor: runner.ParallelRunName,
						Action: model.Scenario{
							Name: runner.SequenceRunName,
							Config: model.ToScenarioConfig(&runner.SequenceCfg{
								Steps: []model.Scenario{
									// create log
									{
										Name: solaris.CreateLogName,
										Config: model.ToScenarioConfig(&solaris.CreateLogCfg{
											Tags: map[string]string{"logName": "foo"},
										}),
									},
									// start 'writersToLog' concurrent writers
									writeConcurrently(writersToLog, appendsToLog, batchSize, msgSize, appendMetricName, appendMsgsMName, appendBytesMName),
									// start 'readers' concurrent readers
									readConcurrently(logReaders, queryStep, queriesNumber, queryMetricName, queryMsgsMName, queryBytesMName),
									// delete log
									{
										Name: solaris.DeleteLogName,
									},
								},
							}),
						},
					}),
				},
				// trace metrics
				{
					Name: runner.MetricsFixRunName,
					Config: model.ToScenarioConfig(&runner.MetricsFixCfg{
						Metrics: []string{appendMetricName, queryMetricName, appendMsgsMName, appendBytesMName, queryMsgsMName, queryBytesMName},
					}),
				},
			},
		}),
	}
}

func writeConcurrently(writersToLog, appendsToLog, batchSize, msgSize int, toMName, msgsRateMName, bytesRateMName string) model.Scenario {
	return model.Scenario{
		Name: runner.RepeatRunName,
		Config: model.ToScenarioConfig(&runner.RepeatCfg{
			Count:    writersToLog,
			Executor: runner.ParallelRunName,
			Action: model.Scenario{
				// start 'appendsToLog' sequential appends
				Name: solaris.AppendRunName,
				Config: model.ToScenarioConfig(&solaris.AppendCfg{
					MessageSize:         msgSize,
					BatchSize:           batchSize,
					Number:              appendsToLog,
					TimeoutMetricName:   toMName,
					MsgsRateMetricName:  msgsRateMName,
					BytesRateMetricName: bytesRateMName,
				}),
			},
		}),
	}
}

func readConcurrently(logReaders, queryStep, queriesNumber int, toMetricName, queryMsgsMName, queryBytesMName string) model.Scenario {
	return model.Scenario{
		Name: runner.RepeatRunName,
		Config: model.ToScenarioConfig(&runner.RepeatCfg{
			Count:    logReaders,
			Executor: runner.ParallelRunName,
			Action: model.Scenario{
				// start sequential queries
				Name: solaris.QueryMsgsRunName,
				Config: model.ToScenarioConfig(&solaris.QueryMsgsCfg{
					Step:                int64(queryStep),
					Number:              queriesNumber,
					TimeoutMetricName:   toMetricName,
					MsgsRateMetricName:  queryMsgsMName,
					BytesRateMetricName: queryBytesMName,
				}),
			},
		}),
	}
}
