package domain

type Amount struct {
	Currency  string
	GrossCent int64
	VatRate   float64
}
