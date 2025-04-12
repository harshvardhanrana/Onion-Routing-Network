package utils

import (
	"crypto/elliptic"
	// "encoding/json"
	// "log"
	"math/big"
	"strings"
	"strconv"

	// ecies "github.com/ecies/go/v2"
)

type retrieve struct {
	CurveParams *elliptic.CurveParams `json:"Curve"`
	MyX         *big.Int              `json:"X"`
	MyY         *big.Int              `json:"Y"`
}

func GetPortAndIP(address string) (uint16, [4]byte) {
	parts := strings.Split(address, ":")
	port, _ := strconv.Atoi(parts[1])
	ipBytes := [4]byte{192, 168, 1, 1}
	return uint16(port), ipBytes
}