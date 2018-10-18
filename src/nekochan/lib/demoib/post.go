package demoib

import (
	"io/ioutil"

	"nekochan/lib/webib0"
)

var _ webib0.PostProvider = (*IBProviderDemo)(nil)

type DemoFile struct{}

func (DemoFile) Write(p []byte) (n int, err error) {
	return ioutil.Discard.Write(p)
}

func (DemoFile) Delete() {
}

type DemoContext struct {
	n int
}

func (c *DemoContext) MakeFile() (webib0.PostFile, error) {
	c.n++
	if c.n > 5 {
		return nil, errTooMuchFiles
	}
	return DemoFile{}, nil
}

func (DemoContext) Release() {
}

func (IBProviderDemo) NewContext() webib0.PostContext {
	return &DemoContext{}
}

func (IBProviderDemo) Submit(p *webib0.PostInfo) error {
	return nil
}
