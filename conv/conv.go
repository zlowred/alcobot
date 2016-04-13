package conv

func NpaToPa(raw int16, offset int16, minValue float64, maxValue float64, minPressure float64, maxPressure float64) float64 {
	return minPressure + (float64(raw)+float64(offset)-minValue)/(maxValue-minValue)*(maxPressure-minPressure)
}

func CtoF(c float64) float64 {
	return c*1.8 + 32
}

func FtoC(f float64) float64 {
	return (f - 32) / 1.8
}

func NpaToC(raw int16) float64 {
	return float64(raw)*200./2048. - 50.
}

func NpaToF(raw int16) float64 {
	return (float64(raw)*200./2048.-50.)*1.8 + 32.
}

func DsToC(raw int16) float64 {
	return float64(raw) * 0.0625
}

func DsToF(raw int16) float64 {
	return float64(raw)*0.1125 + 32.
}

func PaToSg(pa float64, diff float64) float64 {
	return pa / (diff * 9.81)
}
