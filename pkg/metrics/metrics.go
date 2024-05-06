package metrics

import (
	"fmt"
	"sync/atomic"
	"time"

	"golang.org/x/exp/constraints"
)

type (
	Scalar[T Number] struct {
		total atomic.Value //int64
		sum   atomic.Value //T
	}

	String struct {
		total atomic.Value //int64
		value atomic.Value
	}

	Rate struct {
		scale time.Duration
		dur   atomic.Value //time.Duration
		sum   atomic.Value //float64
	}

	Number interface {
		constraints.Float | constraints.Integer
	}

	IntMetricResult struct {
		Total int64 `yaml:"total" json:"total"`
		Sum   int64 `yaml:"sum" json:"sum"`
		Mean  int64 `yaml:"mean" json:"mean"`
	}

	DurationMetricResult struct {
		Total int64         `yaml:"total" json:"total"`
		Sum   time.Duration `yaml:"sum" json:"sum"`
		Mean  time.Duration `yaml:"mean" json:"mean"`
	}

	StringMetricResult struct {
		Total int64  `yaml:"total" json:"total"`
		Sum   string `yaml:"sum" json:"sum"`
	}

	RateMetricResult struct {
		Scale time.Duration `yaml:"scale" json:"scale"`
		Total time.Duration `yaml:"total" json:"total"`
		Sum   float64       `yaml:"sum" json:"sum"`
	}
)

func NewScalar[T Number]() *Scalar[T] {
	s := new(Scalar[T])
	var t T
	s.total.Store(int64(0))
	s.sum.Store(t)
	return s
}

func (s *Scalar[T]) Add(add T) {
	total := s.total.Load().(int64)
	for !s.total.CompareAndSwap(total, total+1) {
		total = s.total.Load().(int64)
	}
	sum := s.sum.Load().(T)
	for !s.sum.CompareAndSwap(sum, sum+add) {
		sum = s.sum.Load().(T)
	}
}

func (s *Scalar[T]) Mean() float64 {
	total := s.total.Load().(int64)
	sum := s.sum.Load().(T)
	return float64(sum) / float64(total)
}

func (s *Scalar[T]) Sum() T {
	return s.sum.Load().(T)
}

func (s *Scalar[T]) Total() int64 {
	return s.total.Load().(int64)
}

func (s *Scalar[T]) Copy() *Scalar[T] {
	var cp Scalar[T]
	cp.total.Store(s.total.Load())
	cp.sum.Store(s.sum.Load())
	return &cp
}

func NewRate(scale time.Duration) *Rate {
	s := new(Rate)
	var t float64
	s.scale = scale
	s.dur.Store(time.Duration(0))
	s.sum.Store(t)
	return s
}

func (s *Rate) Add(value float64, duration time.Duration) {
	total := s.dur.Load().(time.Duration)
	for !s.dur.CompareAndSwap(total, total+duration) {
		total = s.dur.Load().(time.Duration)
	}
	sum := s.sum.Load().(float64)
	for !s.sum.CompareAndSwap(sum, sum+value) {
		sum = s.sum.Load().(float64)
	}
}

func (s *Rate) Rate() float64 {
	total := s.dur.Load().(time.Duration)
	sum := s.sum.Load().(float64)
	return sum / float64(total) * float64(s.scale)
}

func (s *Rate) Sum() float64 {
	return s.sum.Load().(float64)
}

func (s *Rate) Duration() time.Duration {
	return s.dur.Load().(time.Duration)
}

func (s *Rate) Copy() *Rate {
	var cp Rate
	cp.scale = s.scale
	cp.dur.Store(s.dur.Load())
	cp.sum.Store(s.sum.Load())
	return &cp
}

func mean[T Number](data []T) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, d := range data {
		sum += float64(d)
	}
	return sum / float64(len(data))
}

func NewString() *String {
	s := new(String)
	s.total.Store(int64(0))
	s.value.Store("")

	return s
}

func (s *String) Add(value string) {
	total := s.total.Load().(int64)
	for !s.total.CompareAndSwap(total, total+1) {
		total = s.total.Load().(int64)
	}
	lastV := s.value.Load().(string)
	for !s.value.CompareAndSwap(lastV, fmt.Sprintf("%s%s", lastV, value)) {
		lastV = s.value.Load().(string)
	}
}

func (s *String) String() string {
	return s.value.Load().(string)
}

func (s *String) Total() int64 {
	return s.total.Load().(int64)
}

func (s *String) Copy() *String {
	var cp String
	cp.total.Store(s.total.Load())
	cp.value.Store(s.value.Load())
	return &cp
}

func (o1 DurationMetricResult) Merge(o2 DurationMetricResult) DurationMetricResult {
	var res DurationMetricResult
	res.Total = o1.Total + o2.Total
	res.Sum = o1.Sum + o2.Sum
	res.Mean = time.Duration(int64(float64(res.Sum) / float64(res.Total)))
	return res
}

func (o1 IntMetricResult) Merge(o2 IntMetricResult) IntMetricResult {
	var res IntMetricResult
	res.Total = o1.Total + o2.Total
	res.Sum = o1.Sum + o2.Sum
	res.Mean = int64(float64(res.Sum) / float64(res.Total))
	return res
}

func (o1 RateMetricResult) Merge(o2 RateMetricResult) RateMetricResult {
	var res RateMetricResult
	res.Total = o1.Total + o2.Total
	res.Sum = o1.Sum + o2.Sum
	return res
}

func (o1 StringMetricResult) Merge(o2 StringMetricResult) StringMetricResult {
	var res StringMetricResult
	res.Total = o1.Total + o2.Total
	res.Sum = o1.Sum + o2.Sum
	return res
}

func GetDurationMetricResult(metric *Scalar[int64]) DurationMetricResult {
	var metricResult DurationMetricResult
	metricResult.Total = metric.Total()
	metricResult.Mean = time.Duration(int64(metric.Mean()))
	metricResult.Sum = time.Duration(metric.Sum())
	return metricResult
}

func GetStringMetricResult(metric *String) StringMetricResult {
	var metricResult StringMetricResult
	metricResult.Total = metric.Total()
	metricResult.Sum = metric.String()
	return metricResult
}

func GetIntMetricResult(metric *Scalar[int64]) IntMetricResult {
	var metricResult IntMetricResult
	metricResult.Total = metric.Total()
	metricResult.Mean = int64(metric.Mean())
	metricResult.Sum = metric.Sum()
	return metricResult
}

func GetRateMetricResult(metric *Rate) RateMetricResult {
	var metricResult RateMetricResult
	metricResult.Total = metric.Duration()
	metricResult.Scale = metric.scale
	metricResult.Sum = metric.Sum()
	return metricResult
}

func (mr DurationMetricResult) String() string {
	return fmt.Sprintf("{total: %d, sum: %s, mean: %s}", mr.Total, mr.Sum.Round(time.Millisecond), mr.Mean.Round(time.Millisecond))
}

func (mr IntMetricResult) String() string {
	return fmt.Sprintf("{total: %d, sum: %d, mean: %d}", mr.Total, mr.Sum, mr.Mean)
}

func (mr StringMetricResult) String() string {
	return fmt.Sprintf("{total: %d, sum: %s}", mr.Total, mr.Sum)
}

func (mr RateMetricResult) String() string {
	return fmt.Sprintf("{rate: %.2f}", mr.Sum/float64(mr.Total)*float64(mr.Scale))
}
