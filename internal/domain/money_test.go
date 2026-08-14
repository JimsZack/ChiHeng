package domain

import "testing"

func Test_DecimalRoundTrip_when_PreciseValue(t *testing.T) {
	t.Parallel()

	// Given
	input := "123.456789"

	// When
	parsed, err := ParseDecimal(input)

	// Then
	if err != nil || FormatDecimal(parsed) != input {
		t.Fatalf("round trip = %q, %v", FormatDecimal(parsed), err)
	}
}

func Test_Divide_when_DivisorIsZero(t *testing.T) {
	t.Parallel()

	// Given
	const divisor int64 = 0

	// When
	_, err := Divide(Scale, divisor)

	// Then
	if err == nil {
		t.Fatal("expected invalid decimal error")
	}
}
