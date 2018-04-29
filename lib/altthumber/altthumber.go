package altthumber

type AltThumber interface {
	GetAltThumb(fname string, typ string) (alt string, width uint32, height uint32)
}
