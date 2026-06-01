package utils

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

type unit struct {
	name string
	mul  int64
	acc  int
}

// ideally we'd use a sorted map since we want to check for `gwei` suffix before we check `wei` (since one is part
// of the other) however, golang does not support those...
var (
	ether = unit{"ether", 1e18, 18}
	gwei  = unit{"gwei", 1e9, 9}
	wei   = unit{"wei", 1, 1}
)
var units = []unit{ether, gwei, wei}

// ParseUnit parses the provided string and returns a new unit off of it
func ParseUnit(s string) (*big.Int, error) {
	parsed, unit, ok := parse(s)
	if !ok {
		return nil, fmt.Errorf("invalid unit %q", s)
	}

	f := new(big.Float)
	f.SetPrec(236) //  IEEE 754 octuple-precision binary floating-point format: binary256
	f.SetMode(big.ToNearestEven)
	if _, ok := f.SetString(parsed); !ok {
		return nil, fmt.Errorf("invalid value")
	}

	truncInt, _ := f.Int(nil)
	truncInt = new(big.Int).Mul(truncInt, big.NewInt(unit.mul))
	fracStr := strings.Split(fmt.Sprintf("%."+strconv.FormatInt(int64(unit.acc), 10)+"f", f), ".")[1]
	fracStr += strings.Repeat("0", unit.acc-len(fracStr))
	fracInt, _ := new(big.Int).SetString(fracStr, 10)

	return new(big.Int).Add(truncInt, fracInt), nil
}

func parse(value string) (string, unit, bool) {
	value = strings.TrimSpace(value)
	for i := range units {
		if strings.HasSuffix(value, units[i].name) {
			return strings.TrimSpace(strings.ReplaceAll(value, units[i].name, "")), units[i], true
		}
	}
	_, ok := new(big.Float).SetString(value)
	return value, wei, ok
}
