package domain

import "github.com/shopspring/decimal"

// Quality is the code that GSE gives a value, as GSE gives it.
type Quality string

// The codes that GSE gives. GSE documents none of them.
const (
	QualityMeasured    Quality = "ACT"
	QualitySubstituted Quality = "SUB"
	QualityNotReceived Quality = "NRE"
	QualityBLS         Quality = "BLS"
)

// Meaning returns the meaning of the code in English, or "unknown".
func (q Quality) Meaning() string {
	switch q {
	case QualityMeasured:
		return "measured"
	case QualitySubstituted:
		return "substituted by the operator"
	case QualityNotReceived:
		return "not received"
	case QualityBLS:
		return "not documented by GSE"
	default:
		return "unknown"
	}
}

// Measured tells whether the value is a measurement.
func (q Quality) Measured() bool {
	return q == QualityMeasured
}

// Value is one number of GSE, in MW or in Hz, with its quality.
type Value struct {
	Number  decimal.Decimal
	Quality Quality
}

// Direction tells which way power flows across a border.
type Direction string

// The directions of a flow.
const (
	DirectionImport Direction = "import"
	DirectionExport Direction = "export"
	DirectionNone   Direction = "none"
)

// DirectionOf returns the direction of a flow: a positive flow enters Georgia.
func DirectionOf(number decimal.Decimal) Direction {
	switch number.Sign() {
	case 1:
		return DirectionImport
	case -1:
		return DirectionExport
	default:
		return DirectionNone
	}
}
