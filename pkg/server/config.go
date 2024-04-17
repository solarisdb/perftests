package server

import (
	"fmt"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
	"github.com/solarisdb/perftests/pkg/runner/solaris"
)

func GetDefaultConfig() *model.Config {
	return &model.Config{
		Log: model.LoggingConfig{Level: "info"},
		Tests: []model.Test{
			*appendToLogs(10, 3, 10, 3, 2^10),
		},
	}
}

// concurrentLogs - how many logs are written concurrently
// writersToLog - how many writers to one log work concurrently
// appendsToLog -how many appends will be called to one log
// batchSize - how many records are written on one Append call
// msgSize - message size in bytes
func appendToLogs(concurrentLogs, writersToLog, appendsToLog, batchSize, msgSize int) *model.Test {
	return &model.Test{
		Name: fmt.Sprintf("Append to one log"),
		Scenario: model.Scenario{
			Name: runner.SequenceRunName,
			Config: model.ToScenarioConfig(&runner.ParallelCfg{
				Steps: []model.Scenario{
					// connect to solaris
					{
						Name: solaris.ConnectName,
						Config: model.ToScenarioConfig(&solaris.ConnectCfg{
							Address: "localhost:50051",
						}),
					},
					// init metrics
					{
						Name: solaris.MetricsRunName,
						Config: model.ToScenarioConfig(&solaris.MetricsCfg{
							Cmd: solaris.MetricsInit,
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
										{
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
										},
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
							Cmd: solaris.MetricsAppend,
						}),
					},
				},
			}),
		},
	}
}
