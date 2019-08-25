package extthm

func calcDecreaseThumbSize(ow, oh int, tw, th int) (int, int) {
	rw := float64(ow) / float64(tw)
	rh := float64(oh) / float64(th)
	if rw >= rh {
		return int((float64(ow) / rw) + 0.5), int((float64(oh) / rw) + 0.5)
	} else {
		return int((float64(ow) / rh) + 0.5), int((float64(oh) / rh) + 0.5)
	}
}
