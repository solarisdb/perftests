package server

import (
	"fmt"
	"math"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/perftests/pkg/runner/solaris"
)

var defaultAddress = "localhost:50051"
var defaultEnvVarAddress = "PERFTESTS_SOLARIS_ADDRESS"

func GetDefaultConfig() *model.Config {
	// one appender writes to one log 10000 messages by 1K and then
	// one reader reads the log
	test := appendToLogsThenQuery(defaultAddress, defaultEnvVarAddress, 1, 1, 10000, 1, int(math.Pow(float64(2), float64(10))), 1, 100, -1)
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
func appendToLogsThenQuery(svcAddress, envVarAddress string, concurrentLogs, writersToLog, appendsToLog, batchSize, msgSize int,
	logReaders, queryStep, queriesNumber int) *model.Test {
	return &model.Test{
		Name: fmt.Sprintf("Append to %d logs then read it", concurrentLogs),
		Scenario: model.Scenario{
			Name: runner.SequenceRunName,
			Config: model.ToScenarioConfig(&runner.ParallelCfg{
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
						Name: solaris.MetricsRunName,
						Config: model.ToScenarioConfig(&solaris.MetricsCfg{
							Cmds: []solaris.MetricsCmd{solaris.MetricsInit},
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
										writeConcurrently(writersToLog, appendsToLog, batchSize, msgSize),
										// start 'readers' concurrent readers
										readConcurrently(logReaders, queryStep, queriesNumber),
										// delete log
										{
											Name: solaris.DeleteLogName,
										},
									},
								}),
							},
						}),
					},
					// trace append metrics
					{
						Name: solaris.MetricsRunName,
						Config: model.ToScenarioConfig(&solaris.MetricsCfg{
							Cmds: []solaris.MetricsCmd{solaris.MetricsAppend, solaris.MetricsQueryRecords},
						}),
					},
				},
			}),
		},
	}
}

func writeConcurrently(writersToLog, appendsToLog, batchSize, msgSize int) model.Scenario {
	return model.Scenario{
		Name: runner.RepeatRunName,
		Config: model.ToScenarioConfig(&runner.RepeatCfg{
			Count:    writersToLog,
			Executor: runner.ParallelRunName,
			Action: model.Scenario{
				// start 'appendsToLog' sequential appends
				Name: runner.RepeatRunName,
				Config: model.ToScenarioConfig(&runner.RepeatCfg{
					Count:    appendsToLog,
					Executor: runner.SequenceRunName,
					Action: model.Scenario{
						Name: solaris.AppendRunName,
						Config: model.ToScenarioConfig(&solaris.AppendCfg{
							MessageSize: msgSize,
							BatchSize:   batchSize,
						}),
					},
				}),
			},
		}),
	}
}

func readConcurrently(logReaders, queryStep, queriesNumber int) model.Scenario {
	return model.Scenario{
		Name: runner.RepeatRunName,
		Config: model.ToScenarioConfig(&runner.RepeatCfg{
			Count:    logReaders,
			Executor: runner.ParallelRunName,
			Action: model.Scenario{
				// start sequential queries
				Name: solaris.QueryMsgsRunName,
				Config: model.ToScenarioConfig(&solaris.QueryMsgsCfg{
					Step:   int64(queryStep),
					Number: queriesNumber,
				}),
			},
		}),
	}
}
