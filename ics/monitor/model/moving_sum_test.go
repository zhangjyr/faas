package model

import (
	"testing"
)

func NewMovingSumN(n int, window int) *MovingSum {
	movingSum := NewMovingSum(int64(window))
	for i := 1; i <= n; i++  {
		movingSum.Add(float64(i))
	}
	return movingSum
}

func TestSum(t *testing.T) {
	movingSum := NewMovingSumN(1, 5)
	if movingSum.Sum() != 1 {
		t.Logf("wrong sum of 1 with window 5, want: %v, got: %v", 1, movingSum.Sum())
		t.Fail()
	}

	movingSum.Add(2)
	if movingSum.Sum() != 3 {
		t.Logf("wrong sum of 1-2 with window 5, want: %v, got: %v", 3, movingSum.Sum())
		t.Fail()
	}

	movingSum = NewMovingSumN(4, 5)
	if movingSum.Sum() != 10 {
		t.Logf("wrong sum of 1-4 with window 5, want: %v, got: %v", 10, movingSum.Sum())
		t.Fail()
	}

	movingSum.Add(5)
	if movingSum.Sum() != 15 {
		t.Logf("wrong sum of 1-5 with window 5, want: %v, got: %v", 15, movingSum.Sum())
		t.Fail()
	}

	movingSum.Add(6)
	if movingSum.Sum() != 20 {
		t.Logf("wrong sum of 1-6 with window 5, want: %v, got: %v", 20, movingSum.Sum())
		t.Fail()
	}
}

func TestLast(t *testing.T) {
	movingSum := NewMovingSumN(1, 5)
	if movingSum.Last() != 1 {
		t.Logf("wrong last of 1 with window 5, want: %v, got: %v", 1, movingSum.Last())
		t.Fail()
	}

	movingSum.Add(2)
	if movingSum.Last() != 2 {
		t.Logf("wrong last of 1-2 with window 5, want: %v, got: %v", 2, movingSum.Last())
		t.Fail()
	}

	movingSum = NewMovingSumN(4, 5)
	if movingSum.Last() != 4 {
		t.Logf("wrong last of 1-4 with window 5, want: %v, got: %v", 4, movingSum.Last())
		t.Fail()
	}

	movingSum.Add(5)
	if movingSum.Last() != 5 {
		t.Logf("wrong last of 1-5 with window 5, want: %v, got: %v", 5, movingSum.Last())
		t.Fail()
	}

	movingSum.Add(6)
	if movingSum.Last() != 6 {
		t.Logf("wrong last of 1-6 with window 5, want: %v, got: %v", 6, movingSum.Last())
		t.Fail()
	}
}
