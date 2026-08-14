package domain

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var ErrInvalidDecimal = errors.New("invalid decimal")

const Scale int64 = 1_000_000

func ParseDecimal(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, ErrInvalidDecimal
	}
	rational, ok := new(big.Rat).SetString(trimmed)
	if !ok {
		return 0, ErrInvalidDecimal
	}
	scaled := new(big.Rat).Mul(rational, big.NewRat(Scale, 1))
	if !scaled.IsInt() || !scaled.Num().IsInt64() {
		return 0, ErrInvalidDecimal
	}
	return scaled.Num().Int64(), nil
}

func FormatDecimal(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	whole := value / Scale
	fraction := strings.TrimRight(fmt.Sprintf("%06d", value%Scale), "0")
	result := fmt.Sprintf("%d", whole)
	if fraction != "" {
		result += "." + fraction
	}
	if negative {
		return "-" + result
	}
	return result
}

func Multiply(left, right int64) (int64, error) {
	product := new(big.Int).Mul(big.NewInt(left), big.NewInt(right))
	product.Quo(product, big.NewInt(Scale))
	if !product.IsInt64() {
		return 0, ErrInvalidDecimal
	}
	return product.Int64(), nil
}

func Divide(left, right int64) (int64, error) {
	if right == 0 {
		return 0, ErrInvalidDecimal
	}
	quotient := new(big.Int).Mul(big.NewInt(left), big.NewInt(Scale))
	quotient.Quo(quotient, big.NewInt(right))
	if !quotient.IsInt64() {
		return 0, ErrInvalidDecimal
	}
	return quotient.Int64(), nil
}
