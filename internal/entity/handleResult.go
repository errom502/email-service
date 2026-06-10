package entity

type HandleResult int

const (
	ResultAck HandleResult = iota
	ResultRetry
)
