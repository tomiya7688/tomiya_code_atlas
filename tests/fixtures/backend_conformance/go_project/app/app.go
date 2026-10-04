package app

import "example.test/atlasfixture/shared"

// Runner calls a function in another package.
func Runner(input string) shared.Result {
	return shared.Normalize(input)
}

