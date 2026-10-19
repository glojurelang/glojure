package math

import (
	gomath "math"
	"reflect"
	"testing"

	"github.com/glojurelang/glojure/pkg/lang"
)

func TestFloatFunctionsAcceptRatios(t *testing.T) {
	floor := fn1Float64(func(x float64) float64 { return x })
	if got, want := lang.Apply1(floor, lang.NewRatio(3, 2)), 1.5; got != want {
		t.Fatalf("ratio conversion = %v, want %v", got, want)
	}
}

func TestMathFloatingPointBoundaries(t *testing.T) {
	negativeZero := gomath.Copysign(0, -1)
	cases := []struct {
		name string
		run  func() any
		want any
	}{
		{"ulp-zero", func() any { return Ulp(0.0) }, gomath.SmallestNonzeroFloat64},
		{"ulp-subnormal", func() any { return Ulp(gomath.SmallestNonzeroFloat64) }, gomath.SmallestNonzeroFloat64},
		{"ulp-max", func() any { return Ulp(gomath.MaxFloat64) }, gomath.Ldexp(1, 971)},
		{"ulp-infinity", func() any { return Ulp(gomath.Inf(-1)) }, gomath.Inf(1)},
		{"ulp-float", func() any { return Ulp(float32(1)) }, float32(gomath.Ldexp(1, -23))},
		{"ulp-float-zero", func() any { return Ulp(float32(0)) }, float32(gomath.SmallestNonzeroFloat32)},
		{"ulp-float-max", func() any { return Ulp(float32(gomath.MaxFloat32)) }, float32(gomath.Ldexp(1, 104))},
		{"ulp-float-infinity", func() any { return Ulp(float32(gomath.Inf(-1))) }, float32(gomath.Inf(1))},
		{"exponent-zero", func() any { return GetExponent(0.0) }, int32(-1023)},
		{"exponent-subnormal", func() any { return GetExponent(gomath.SmallestNonzeroFloat64) }, int32(-1023)},
		{"exponent-infinity", func() any { return GetExponent(gomath.Inf(-1)) }, int32(1024)},
		{"exponent-float-subnormal", func() any { return GetExponent(float32(gomath.SmallestNonzeroFloat32)) }, int32(-127)},
		{"exponent-float-infinity", func() any { return GetExponent(float32(gomath.Inf(1))) }, int32(128)},
		{"after-zero-sign", func() any { return NextAfter(0.0, negativeZero) }, negativeZero},
		{"after-float-zero-sign", func() any { return NextAfter(float32(0), negativeZero) }, float32(negativeZero)},
		{"after-float-direction", func() any { return NextAfter(float32(1), 1+gomath.Ldexp(1, -40)) }, gomath.Nextafter32(1, 2)},
		{"after-float-infinity", func() any { return NextAfter(float32(gomath.Inf(1)), 0.0) }, float32(gomath.MaxFloat32)},
		{"up-zero", func() any { return NextUp(negativeZero) }, gomath.SmallestNonzeroFloat64},
		{"down-zero", func() any { return NextDown(0.0) }, -gomath.SmallestNonzeroFloat64},
		{"up-float-zero", func() any { return NextUp(float32(negativeZero)) }, float32(gomath.SmallestNonzeroFloat32)},
		{"down-float-zero", func() any { return NextDown(float32(0)) }, float32(-gomath.SmallestNonzeroFloat32)},
		{"scale-zero-sign", func() any { return Scalb(negativeZero, int32(10)) }, negativeZero},
		{"scale-subnormal", func() any { return Scalb(1.0, int32(-1074)) }, gomath.SmallestNonzeroFloat64},
		{"scale-float-subnormal", func() any { return Scalb(float32(1), int32(-149)) }, float32(gomath.SmallestNonzeroFloat32)},
		{"scale-float-overflow", func() any { return Scalb(float32(1), int32(2147483647)) }, float32(gomath.Inf(1))},
		{"scale-underflow", func() any { return Scalb(1.0, int32(-2147483648)) }, 0.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.run()
			if reflect.TypeOf(got) != reflect.TypeOf(tc.want) {
				t.Fatalf("type = %T, want %T", got, tc.want)
			}
			switch want := tc.want.(type) {
			case float64:
				if gomath.Float64bits(got.(float64)) == gomath.Float64bits(want) {
					return
				}
			case float32:
				if gomath.Float32bits(got.(float32)) == gomath.Float32bits(want) {
					return
				}
			default:
				if got == tc.want {
					return
				}
			}
			t.Fatalf("got %v, want %v", got, tc.want)
		})
	}
}

func TestMathNaN(t *testing.T) {
	for name, call := range map[string]func(any) any{
		"ulp": Ulp, "next-up": NextUp, "next-down": NextDown,
		"next-after": func(x any) any { return NextAfter(x, 1.0) },
		"scale":      func(x any) any { return Scalb(x, int32(2)) },
	} {
		t.Run(name, func(t *testing.T) {
			if got := call(gomath.NaN()); !gomath.IsNaN(got.(float64)) {
				t.Fatalf("got %v, want NaN", got)
			}
			if got := call(float32(gomath.NaN())); !gomath.IsNaN(float64(got.(float32))) {
				t.Fatalf("got %v, want float32 NaN", got)
			}
		})
	}
	if got := GetExponent(gomath.NaN()); got != 1024 {
		t.Fatalf("NaN exponent = %d, want 1024", got)
	}
	if got := GetExponent(float32(gomath.NaN())); got != 128 {
		t.Fatalf("float32 NaN exponent = %d, want 128", got)
	}
}
