package server

import (
	"fmt"

	"github.com/solarisdb/perftests/pkg/model"
	"github.com/solarisdb/perftests/pkg/runner"
)

var tenantID = "og2ch03a68h"

func GetDefaultConfig() *model.Config {
	lcfg := model.LoggingConfig{Level: "trace"}
	return &model.Config{
		Log: lcfg,
		//Environment: configs.DevEnvName,
		Tests: []model.Test{
			testContextPropagation(),
			//testErrors(),
		},
	}
}

func testErrors() model.Test {
	return model.Test{
		Name: fmt.Sprintf("Test errors"),
		Scenario: model.Scenario{
			Name: runner.SequenceRunName,
			Config: model.ToScenarioConfig(&runner.SequenceCfg{
				Steps: []model.Scenario{
					{
						Name: runner.ParallelRunName,
						Config: model.ToScenarioConfig(&runner.ParallelCfg{
							Steps: []model.Scenario{
								{
									Name: runner.SequenceRunName,
									Config: model.ToScenarioConfig(&runner.SequenceCfg{
										Steps: []model.Scenario{
											{
												Name: runner.RepeatRunName,
												Config: model.ToScenarioConfig(&runner.RepeatCfg{
													Executor:   runner.ParallelRunName,
													Count:      10,
													SkipErrors: true,
													Action: model.Scenario{
														Name: runner.ErrorRunName,
														Config: model.ToScenarioConfig(&runner.ErrorCfg{
															Error: "Hello from error test",
														}),
													},
												}),
											},
										},
									}),
								},
							},
						}),
					},
				},
			}),
		},
	}
}

// test counts Pause Runners and print total count at the end, 33 pauses is an expected count
func testContextPropagation() model.Test {
	return model.Test{
		Name: fmt.Sprintf("Test context propagation"),
		Scenario: model.Scenario{
			Name: runner.SequenceRunName,
			Config: model.ToScenarioConfig(&runner.SequenceCfg{
				Steps: []model.Scenario{
					{
						Name: runner.ParallelRunName,
						Config: model.ToScenarioConfig(&runner.ParallelCfg{
							Steps: []model.Scenario{
								{

									Name: runner.SequenceRunName,
									Config: model.ToScenarioConfig(&runner.SequenceCfg{
										Steps: []model.Scenario{
											{
												Name: runner.PauseRunName,
												Config: model.ToScenarioConfig(&runner.PauseCfg{
													Value: "0s",
												}),
											},
										},
									}),
								},
								{

									Name: runner.SequenceRunName,
									Config: model.ToScenarioConfig(&runner.SequenceCfg{
										Steps: []model.Scenario{
											{
												Name: runner.PauseRunName,
												Config: model.ToScenarioConfig(&runner.PauseCfg{
													Value: "0s",
												}),
											},
										},
									}),
								},
								{

									Name: runner.RepeatRunName,
									Config: model.ToScenarioConfig(&runner.RepeatCfg{
										Executor:   runner.SequenceRunName,
										Count:      10,
										SkipErrors: false,
										Action: model.Scenario{
											Name: runner.PauseRunName,
											Config: model.ToScenarioConfig(&runner.PauseCfg{
												Value: "0s",
											}),
										},
									}),
								},
								{

									Name: runner.RepeatRunName,
									Config: model.ToScenarioConfig(&runner.RepeatCfg{
										Executor:   runner.ParallelRunName,
										Count:      10,
										SkipErrors: false,
										Action: model.Scenario{
											Name: runner.PauseRunName,
											Config: model.ToScenarioConfig(&runner.PauseCfg{
												Value: "0s",
											}),
										},
									}),
								},
							},
						}),
					},
					{
						Name: runner.PauseRunName,
						Config: model.ToScenarioConfig(&runner.PauseCfg{
							Value: "0s",
						}),
					},
					{
						Name: runner.SequenceRunName,
						Config: model.ToScenarioConfig(&runner.SequenceCfg{
							Steps: []model.Scenario{
								{
									Name: runner.RepeatRunName,
									Config: model.ToScenarioConfig(&runner.RepeatCfg{
										Executor:   runner.SequenceRunName,
										Count:      10,
										SkipErrors: false,
										Action: model.Scenario{
											Name: runner.PauseRunName,
											Config: model.ToScenarioConfig(&runner.PauseCfg{
												Value: "0s",
											}),
										},
									}),
								},
							},
						}),
					},
				},
			}),
		},
	}
}
