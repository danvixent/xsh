package main

import "os"

func (p *Plan) InterceptOSSignal(sig os.Signal) {
	close(p.stop)
}
