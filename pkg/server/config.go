package server

import (
	"fmt"
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

func GetDefaultConfig() *model.Config {
	//return buildAppendToManyLogsTests()
	// one appender writes to one log 10000 messages by 1K and then
	// one reader reads the log
	test := appendToLogsThenQueryTest(defaultEnvRunID, defaultAddress, defaultEnvVarAddress, 2500, 1, 10, 1, int(math.Pow(float64(2), float64(10))), 1, 100, -1)
	return &model.Config{
		Log: model.LoggingConfig{Level: "info"},
		Tests: []model.Test{
			*test,
		},
	}
}

func cleanupCluster() *model.Test {
	scenario := clusterCleanup(defaultEnvRunID, defaultAddress, defaultEnvVarAddress)
	return &model.Test{
		Name:     fmt.Sprintf("Cleanup cluster %s", defaultEnvRunID),
		Scenario: *scenario,
	}
}

func buildAppendToManyLogsTests() *model.Config {
	//test0 := cleanupCluster()
	concLogs := 100000
	writers := 1
	data := 10 * oneGB
	// append to 10K logs (one log one writer), write 10GB data by 1kB messages
	test1 := appendToManyLogs(concLogs, writers, data, 1*oneKb)
	// append to 10K logs (one log one writer), write 10MB data by 10kB messages
	test2 := appendToManyLogs(concLogs, writers, data, 10*oneKb)
	// append to 10K logs (one log one writer), write 10MB data by 100kB messages
	test3 := appendToManyLogs(concLogs, writers, data, 100*oneKb)
	return &model.Config{
		Log: model.LoggingConfig{Level: defLogLevel},
		Tests: []model.Test{
			//*test0,
			*test1,
			*test2,
			*test3,
		},
	}
}

// appendToManyLogs appends to concurrentLogs logs (for one log one writer), write totalSize data by oneStep size of messages
func appendToManyLogs(concurrentLogs, writers int, totalSize, oneStep int) *model.Test {
	appendsToLog := totalSize / oneStep
	test := appendToLogsThenQueryTest(defaultEnvRunID, defaultAddress, defaultEnvVarAddress, concurrentLogs, writers, appendsToLog, 1, oneStep, 0, 100, -1)
	test.Name = fmt.Sprintf("Append to %d logs, write %s to each one, one message size %s", concurrentLogs, humanReadableSize(totalSize), humanReadableSize(oneStep))
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

var sizes = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}

func humanReadableSize(origSize int) string {
	base := 1024.0
	unitsLimit := len(sizes)
	i := 0
	size := float64(origSize)
	for size >= base && i < unitsLimit {
		size = size / base
		i++
	}

	f := "%.0f %s"
	if i > 1 {
		f = "%.2f %s"
	}

	return fmt.Sprintf(f, size, sizes[i])
}
