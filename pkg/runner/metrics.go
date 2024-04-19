package runner

import (
	"fmt"
	"sync/atomic"

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

	Number interface {
		constraints.Float | constraints.Integer
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
