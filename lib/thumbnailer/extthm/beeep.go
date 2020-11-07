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

// only if smaller
func calcDecreaseThumbSizeOIS(ow, oh int, tw, th int) (int, int) {
	rw := float64(ow) / float64(tw)
	rh := float64(oh) / float64(th)
	if rw >= rh {
		if rw <= 1 {
			return ow, oh
		}
		return int((float64(ow) / rw) + 0.5), int((float64(oh) / rw) + 0.5)
	} else {
		if rh <= 1 {
			return ow, oh
		}
		return int((float64(ow) / rh) + 0.5), int((float64(oh) / rh) + 0.5)
	}
}
