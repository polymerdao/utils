// Package utils holds those functions used across  the code base that don't have a better home than this
package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/polymerdao/utils/common"
	"golang.org/x/crypto/sha3"
)

// Bytes32ToStr converts a bytes32 object into a string
func Bytes32ToStr(b common.Hash) string {
	return strings.TrimRight(string(b[:]), "\x00")
}

// StrToBytes32 converts a string object into a bytes32
func StrToBytes32(s string) (h common.Hash) {
	copy(h[:], []byte(s))
	return h
}

// Keccak256 returns a keccak256 hash of the provided data
func Keccak256(data ...[]byte) common.Hash {
	hasher := sha3.NewLegacyKeccak256()
	for i := range data {
		hasher.Write(data[i])
	}
	return common.BytesToHash(hasher.Sum(nil))
}

// Marshal implements a json marshalling of all the fields received in the args list. The list
// must be of the form ["key0", value0, "key1", value1...] with even number of elements.
func Marshal(args ...any) ([]byte, error) {
	if len(args)&1 != 0 {
		panic(fmt.Errorf("args must be an even list of arguments. got %d", len(args)))
	}
	m := make(map[string]any)
	for i := 0; i < len(args); i += 2 {
		// if args[i] is not a string, let it panic. This is a bug
		k := args[i].(string)
		// if the arg is []byte, hash it to make it look "nice"
		if bz, ok := args[i+1].([]byte); ok {
			m[k] = Keccak256(bz)
			continue
		}
		// if the arg is string longer than a common.Hash() as string, hash it too
		if bz, ok := args[i+1].(string); ok && len(bz) > 66 {
			m[k] = Keccak256([]byte(bz))
			continue
		}
		m[k] = args[i+1]
	}
	return json.Marshal(m)
}

// GetFuncName returns the function name of caller
func GetFuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	details := runtime.FuncForPC(pc)
	if details == nil {
		return ""
	}
	name := details.Name()
	index := strings.LastIndex(name, ".")
	if index >= 0 {
		return name[index+1:]
	}
	return name
}

// SleepContext sleeps for the specified duration or until the context is done, whichever comes first
func SleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
