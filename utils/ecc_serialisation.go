package utils

import (
	"crypto/elliptic"
	// "encoding/json"
	// "log"
	"math/big"

	// ecies "github.com/ecies/go/v2"
)

type retrieve struct {
	CurveParams *elliptic.CurveParams `json:"Curve"`
	MyX         *big.Int              `json:"X"`
	MyY         *big.Int              `json:"Y"`
}

