package webib0

type IBProvider interface {
	IBGetBoardList(*IBBoardList) (error, int)
	IBGetThreadListPage(*IBThreadListPage, string, uint32) (error, int)
	IBGetThreadCatalog(*IBThreadCatalog, string) (error, int)
	IBGetThread(*IBThreadPage, string, string) (error, int)
}
