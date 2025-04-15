package utils

import (
	"errors"
	"google.golang.org/grpc/status"
)
	

var (
	ErrCircuitNotFound = errors.New("circuit ID not found")
)


func IsEqual(err error, targetErr error) bool {
	if err == targetErr {   // to handle the case with targetErr = nil (ErrSuccess)
		return true
	}
	if targetErr == nil {  // if targetErr == nil, err != nil
		return false
	}
    if errors.Is(err, targetErr) {
        return true
    }
    s, ok := status.FromError(err)
    return ok && (s.Message() == targetErr.Error())   
}