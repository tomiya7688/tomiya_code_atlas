package partition

func AlphaStart() {
	AlphaMiddle()
}

func AlphaMiddle() {
	AlphaWrite()
}

func AlphaWrite() {
	SharedLogger("alpha")
}

func BetaStart() {
	BetaMiddle()
}

func BetaMiddle() {
	BetaWrite()
}

func BetaWrite() {
	SharedLogger("beta")
}

func GammaStart() {
	GammaMiddle()
}

func GammaMiddle() {
	GammaWrite()
}

func GammaWrite() {
	SharedLogger("gamma")
}

func SharedLogger(message string) {}
