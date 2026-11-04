package recursion

func alpha() {
	beta()
}

func beta() {
	alpha()
	self()
}

func self() {
	self()
}
