package lang

import "testing"

func TestShortCastRejectsFractionalValuesOutsideRange(t *testing.T) {
	for _, value := range []float64{-32768.000001, 32767.000001} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("ShortCast(%v) did not panic", value)
				}
			}()
			ShortCast(value)
		}()
	}
}
