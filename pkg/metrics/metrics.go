package metrics

import (
	"fmt"
	"github.com/solarisdb/solaris/golibs/container"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/constraints"
)

type (
	// Scalar is a gauge metric that could be increased/decreased
	Scalar[T Number] struct {
		total atomic.Value //int64
		sum   atomic.Value //T
	}

	String struct {
		total atomic.Value //int64
		value atomic.Value
	}

	// Rate is a histogram metric
	Rate struct {
		scale   time.Duration
		samples []*RateSample

		lock sync.Mutex
	}

	RateSample struct {
		Start    int64         `yaml:"s" json:"s"`
		Value    float64       `yaml:"v,omitempty" json:"v,omitempty"`
		Duration time.Duration `yaml:"d,omitempty" json:"d,omitempty"`
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
		Scale   time.Duration `yaml:"scale" json:"scale"`
		Samples []*RateSample `yaml:"samples" json:"samples"`
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
	s.scale = scale
	return s
}

func (s *Rate) Add(value float64, duration time.Duration) {
	end := time.Now()
	start := end.Add(-duration)
	newSamples := s.calc(value, start, end)
	nIndx := len(newSamples)
	if nIndx == 0 {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.samples = mergeSamples(s.samples, newSamples, s.scale)
}

func mergeSamples(to, from []*RateSample, scale time.Duration) []*RateSample {
	fromIndx := len(from)
	toIndx := len(to)
	for {
		if fromIndx == 0 {
			break
		}
		if toIndx == 0 {
			to = append(from[:fromIndx], to...)
			break
		}
		if to[toIndx-1].Start == from[fromIndx-1].Start {
			merged := RateSample{
				Start:    to[toIndx-1].Start,
				Value:    to[toIndx-1].Value + from[fromIndx-1].Value,
				Duration: min(to[toIndx-1].Duration+from[fromIndx-1].Duration, scale),
			}
			to[toIndx-1] = &merged
			fromIndx--
			toIndx--
		} else if to[toIndx-1].Start < (from[fromIndx-1].Start) {
			to = append(to[:toIndx], to[toIndx-1:]...)
			to[toIndx] = from[fromIndx-1]
			fromIndx--
		} else {
			toIndx--
		}
	}
	return to
}

func (s *Rate) calc(value float64, start, end time.Time) []*RateSample {
	sampSt := start.Truncate(s.scale)
	sampEnd := end.Truncate(s.scale)
	var result []*RateSample
	allDuration := end.Sub(start)
	for i := sampSt; !i.After(sampEnd); i = i.Add(s.scale) {
		left := maxTime(i, start)
		right := minTime(i.Add(s.scale), end)
		sampleValDur := right.Sub(left)
		part := 1.0
		if sampleValDur != allDuration {
			part = float64(sampleValDur.Nanoseconds()) / float64(allDuration.Nanoseconds())
		}
		result = append(result, &RateSample{
			Start:    i.UnixNano() / int64(s.scale),
			Value:    value * part,
			Duration: sampleValDur,
		})
	}
	return result
}

func minTime(t1, t2 time.Time) time.Time {
	if t1.Before(t2) {
		return t1
	}
	return t2
}
func maxTime(t1, t2 time.Time) time.Time {
	if t1.Before(t2) {
		return t2
	}
	return t1
}

func (s *Rate) Rate() float64 {
	var sum float64
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.samples) == 0 {
		return math.NaN()
	}
	withDur := s.samples[0].Duration > 0
	for _, sm := range s.samples {
		if withDur {
			sum += sm.Value / float64(sm.Duration) * float64(s.scale)
		} else {
			sum += sm.Value
		}
	}
	return sum / float64(len(s.samples))
}

func (s *Rate) IntervalRate() float64 {
	var sum float64
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.samples) == 0 {
		return math.NaN()
	}
	withDur := s.samples[0].Duration > 0
	for _, sm := range s.samples {
		if withDur {
			sum += sm.Value / float64(sm.Duration) * float64(s.scale)
		} else {
			sum += sm.Value
		}
	}
	distance := max(1, s.samples[len(s.samples)-1].Start-s.samples[0].Start)
	return sum / float64(distance)
}

func (s *Rate) Copy() *Rate {
	var cp Rate
	cp.scale = s.scale
	cp.samples = container.SliceCopy(s.samples)
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
	res.Samples = mergeSamples(container.SliceCopy(o1.Samples), container.SliceCopy(o2.Samples), o1.Scale)
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
	metricResult.Samples = container.SliceCopy(metric.samples)
	metricResult.Scale = metric.scale
	return metricResult
}

func FromRateMetricResult(result RateMetricResult) *Rate {
	var rate Rate
	rate.samples = container.SliceCopy(result.Samples)
	rate.scale = result.Scale
	return &rate
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
	return FromRateMetricResult(mr).String()
}

func (r Rate) String() string {
	var scaleStr string
	switch r.scale {
	case time.Second:
		scaleStr = "in sec"
	case time.Millisecond:
		scaleStr = "in millis"
	case time.Minute:
		scaleStr = "in min"
	}
	return fmt.Sprintf("{mean rate: %.2f %s, mean interval rate: %.2f %s}", r.Rate(), scaleStr, r.IntervalRate(), scaleStr)
}
