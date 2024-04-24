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

const appendToMetricName = "AppendTimeout"
const queryToMetricName = "QueryTimeout"

func GetDefaultConfig() *model.Config {
	// one appender writes to one log 10000 messages by 1K and then
	// one reader reads the log
	test := appendToLogsThenQueryTest(defaultEnvRunID, defaultAddress, defaultEnvVarAddress, 1, 1, 100, 1, int(math.Pow(float64(2), float64(10))), 1, 100, -1)
	return &model.Config{
		Log: model.LoggingConfig{Level: "info"},
		Tests: map[string]model.Test{
			test.Name: *test,
		},
	}
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
	return &model.Test{
		Name:     fmt.Sprintf("Append to %d logs then read it", concurrentLogs),
		Scenario: *clusterRun(runID, svcAddress, envVarAddress, scenario),
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

func appendToLogsThenQueryScenario(svcAddress, envVarAddress string, concurrentLogs, writersToLog, appendsToLog, batchSize, msgSize int,
	logReaders, queryStep, queriesNumber int) *model.Scenario {
	appendMetricName := appendToMetricName
	queryMetricName := queryToMetricName
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
									writeConcurrently(writersToLog, appendsToLog, batchSize, msgSize, appendMetricName),
									// start 'readers' concurrent readers
									readConcurrently(logReaders, queryStep, queriesNumber, queryMetricName),
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
						Metrics: []string{appendMetricName, queryMetricName},
					}),
				},
			},
		}),
	}
}

func writeConcurrently(writersToLog, appendsToLog, batchSize, msgSize int, metricName string) model.Scenario {
	return model.Scenario{
		Name: runner.RepeatRunName,
		Config: model.ToScenarioConfig(&runner.RepeatCfg{
			Count:    writersToLog,
			Executor: runner.ParallelRunName,
			Action: model.Scenario{
				// start 'appendsToLog' sequential appends
				Name: solaris.AppendRunName,
				Config: model.ToScenarioConfig(&solaris.AppendCfg{
					MessageSize:       msgSize,
					BatchSize:         batchSize,
					Number:            appendsToLog,
					TimeoutMetricName: metricName,
				}),
			},
		}),
	}
}

func readConcurrently(logReaders, queryStep, queriesNumber int, metricName string) model.Scenario {
	return model.Scenario{
		Name: runner.RepeatRunName,
		Config: model.ToScenarioConfig(&runner.RepeatCfg{
			Count:    logReaders,
			Executor: runner.ParallelRunName,
			Action: model.Scenario{
				// start sequential queries
				Name: solaris.QueryMsgsRunName,
				Config: model.ToScenarioConfig(&solaris.QueryMsgsCfg{
					Step:              int64(queryStep),
					Number:            queriesNumber,
					TimeoutMetricName: metricName,
				}),
			},
		}),
	}
}
