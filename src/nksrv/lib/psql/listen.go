package psql

import (
	"errors"
	"time"

	. "nksrv/lib/logx"

	"github.com/lib/pq"
)

func (p *PSQL) listenEventCallback(et pq.ListenerEventType, err error) {
	switch et {
	case pq.ListenerEventConnected:
		p.log.LogPrint(NOTICE, "LISTEN connected")
	case pq.ListenerEventDisconnected:
		p.log.LogPrint(NOTICE, "LISTEN disconnected: %v", err)
	case pq.ListenerEventReconnected:
		p.log.LogPrint(NOTICE, "LISTEN reconnected")
	case pq.ListenerEventConnectionAttemptFailed:
		p.log.LogPrint(NOTICE, "LISTEN failed reconnect: %v", err)
	}
}

func (p *PSQL) listenProcessor(cn <-chan *pq.Notification) {
	for e := range cn {
		p.limMu.Lock()
		if e != nil {
			p.log.LogPrint(DEBUG, "LISTEN notif[%s] %q", e.Channel, e.Extra)
			p.lim[e.Channel](e.Extra, false)
		} else {
			p.log.LogPrint(DEBUG, "LISTEN rst notif")
			for _, f := range p.lim {
				f("", true)
			}
		}
		p.limMu.Unlock()
	}
	// quit when channel closed
}

type ListenCB func(e string, rst bool)

func (p *PSQL) Listen(n string, f ListenCB) error {
	p.lii.Do(func() {
		p.li = pq.NewListener(
			p.connstr,
			500*time.Millisecond,
			15*time.Second,
			func(et pq.ListenerEventType, err error) {
				p.listenEventCallback(et, err)
			})
		go p.listenProcessor(p.li.Notify)
	})

	if p.li == nil {
		return errors.New("PSQL closing")
	}

	p.limMu.Lock()
	defer p.limMu.Unlock()

	if _, ee := p.lim[n]; ee {
		return errors.New("something already listens on this channel")
	}

	p.lim[n] = f

	return nil
}
