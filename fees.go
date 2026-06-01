package utils

import "math/big"

var (
	hundred = big.NewInt(100)
	two     = big.NewInt(2)
)

// CalcGasFeeCap deterministically computes the recommended gas fee cap given
// the base fee and gasTipCap. The resulting gasFeeCap is equal to: gasTipCap + 2*baseFee
func CalcGasFeeCap(baseFee, gasTipCap *big.Int) *big.Int {
	return new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(baseFee, two),
	)
}

// AddMargin adds a margin (percentage) to base using the following formula: base + ( base * margin ) / 100
func AddMargin(base, margin *big.Int) *big.Int {
	return new(big.Int).Add(
		base,
		new(big.Int).Div(
			new(big.Int).Mul(base, margin),
			hundred,
		),
	)
}

// Max returns the max of two big ints. Does not check for nullity
func Max(a, b *big.Int) *big.Int {
	if a.Cmp(b) > 0 {
		return a
	}
	return b
}
