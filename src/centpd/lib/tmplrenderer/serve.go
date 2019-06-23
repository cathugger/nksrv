package tmplrenderer

import (
	"net/http"

	ib0 "centpd/lib/webib0"
)

func (tr *TmplRenderer) ServeBoardList(
	w http.ResponseWriter, r *http.Request) {

	l := &struct {
		D ib0.IBBoardList
		N *NodeInfo
		R *TmplRenderer
	}{
		N: &tr.ni,
		R: tr,
	}

	err, code := tr.p.IBGetBoardList(&l.D)
	if err != nil {
		ctx := struct {
			Code int
			Err  error
		}{
			code,
			err,
		}
		tr.outTmplP(w, ptmplBoardListErr, code, ctx)
		return
	}
	tr.outTmplP(w, ptmplBoardList, 200, l)
}

func (tr *TmplRenderer) ServeThreadListPage(
	w http.ResponseWriter, r *http.Request, board string, page uint32) {

	l := &struct {
		D ib0.IBThreadListPage
		N *NodeInfo
		R *TmplRenderer
		C string
	}{
		N: &tr.ni,
		R: tr,
		C: tr.newCaptchaKey(),
	}

	err, code := tr.p.IBGetThreadListPage(&l.D, board, page)
	if err != nil {
		ctx := struct {
			Code  int
			Err   error
			Board string
			Page  uint32
		}{
			code,
			err,
			board,
			page,
		}
		tr.outTmplP(w, ptmplThreadListPageErr, code, ctx)
		return
	}
	if !l.D.HasBackRefs {
		for i := range l.D.Threads {
			ib0.ProcessBackReferences(l.D.Board.Name, &l.D.Threads[i].IBCommonThread)
		}
		l.D.HasBackRefs = true
	}
	tr.outTmplP(w, ptmplThreadListPage, 200, l)
}

func (tr *TmplRenderer) ServeOverboardPage(
	w http.ResponseWriter, r *http.Request, page uint32) {

	l := &struct {
		D ib0.IBOverboardPage
		N *NodeInfo
		R *TmplRenderer
	}{
		N: &tr.ni,
		R: tr,
	}

	err, code := tr.p.IBGetOverboardPage(&l.D, page)
	if err != nil {
		ctx := struct {
			Code int
			Err  error
			Page uint32
		}{
			code,
			err,
			page,
		}
		tr.outTmplP(w, ptmplOverboardPageErr, code, ctx)
		return
	}
	if !l.D.HasBackRefs {
		for i := range l.D.Threads {
			ib0.ProcessBackReferences(
				l.D.Threads[i].BoardName, &l.D.Threads[i].IBCommonThread)
		}
		l.D.HasBackRefs = true
	}
	tr.outTmplP(w, ptmplOverboardPage, 200, l)
}

func (tr *TmplRenderer) ServeThread(
	w http.ResponseWriter, r *http.Request, board, thread string) {

	l := &struct {
		D ib0.IBThreadPage
		N *NodeInfo
		R *TmplRenderer
		C string
	}{
		N: &tr.ni,
		R: tr,
		C: tr.newCaptchaKey(),
	}

	err, code := tr.p.IBGetThread(&l.D, board, thread)
	if err != nil {
		ctx := struct {
			Code   int
			Err    error
			Board  string
			Thread string
		}{
			code,
			err,
			board,
			thread,
		}
		tr.outTmplP(w, ptmplThreadErr, code, ctx)
		return
	}
	if !l.D.HasBackRefs {
		ib0.ProcessBackReferences(l.D.Board.Name, &l.D.IBCommonThread)
		l.D.HasBackRefs = true
	}
	tr.outTmplP(w, ptmplThread, 200, l)
}

func (tr *TmplRenderer) ServeThreadCatalog(
	w http.ResponseWriter, r *http.Request, board string) {

	l := &struct {
		D ib0.IBThreadCatalog
		N *NodeInfo
		R *TmplRenderer
	}{
		N: &tr.ni,
		R: tr,
	}

	err, code := tr.p.IBGetThreadCatalog(&l.D, board)
	if err != nil {
		ctx := struct {
			Code  int
			Err   error
			Board string
		}{
			code,
			err,
			board,
		}
		tr.outTmplP(w, ptmplThreadCatalogErr, code, ctx)
		return
	}
	tr.outTmplP(w, ptmplThreadCatalog, 200, l)
}
