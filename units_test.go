package utils

import (
	"math/big"
	"strings"
	"testing"
)

func TestUnitParse(t *testing.T) {
	for _, tc := range []struct {
		str string
		exp *big.Int
	}{{
		str: "1ether",
		exp: big.NewInt(1000000000000000000),
	}, {
		str: "0.5ether",
		exp: big.NewInt(500000000000000000),
	}, {
		str: " 0.5 ether ",
		exp: big.NewInt(500000000000000000),
	}, {
		str: "0.5gwei",
		exp: big.NewInt(500000000),
	}, {
		str: "5wei",
		exp: big.NewInt(5),
	}, {
		str: "59",
		exp: big.NewInt(59),
	}, {
		str: "-3wei",
		exp: big.NewInt(-3),
	}, {
		str: "0.000000000000000001ether",
		exp: big.NewInt(1),
	}, {
		str: "0.00025ether",
		exp: big.NewInt(250000000000000),
	}} {
		t.Run(tc.str, func(t *testing.T) {
			u, err := ParseUnit(tc.str)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if u.Cmp(tc.exp) != 0 {
				t.Fatalf("got %s, want %s", u, tc.exp)
			}
		})
	}
}

func TestUnitErrors(t *testing.T) {
	for _, tc := range []struct {
		str string
		err string
	}{{
		str: "1eth",
		err: `invalid unit`,
	}, {
		str: "1foo",
		err: `invalid unit`,
	}, {
		str: "0,1ether",
		err: `invalid value`,
	}, {
		str: "",
		err: `invalid unit`,
	}, {
		str: "ether",
		err: `invalid value`,
	}} {
		t.Run(tc.str, func(t *testing.T) {
			u, err := ParseUnit(tc.str)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.err)
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.err)
			}
			if u != nil {
				t.Fatalf("expected nil result, got %v", u)
			}
		})
	}
}
