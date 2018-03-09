package demoib

import (
	"../webib0"
	"errors"
	"net/http"
)

type IBProviderDemo struct{}

var _ webib0.IBProvider = (*IBProviderDemo)(nil)

var testNodeInfo = webib0.IBNodeInfo{
	Name: "testnode",
}

var testBoardInfo = webib0.IBBoardInfo{
	Name:        "test",
	Description: "board for testing",
	Info:        "nothing of value visible there",
}
var (
	testThumb1 = webib0.IBThumbInfo{
		ID:     "1.png.jpg",
		Width:  128,
		Height: 128,
	}
	testFile1 = webib0.IBFileInfo{
		ID:       "1.png",
		Type:     "image",
		Thumb:    testThumb1,
		Original: "original test file.png",
		Size:     2048,
		Options: map[string]interface{}{
			"width":  480,
			"height": 480,
		},
	}
	testThumb2 = webib0.IBThumbInfo{
		ID:     "2.jpg.jpg",
		Width:  128,
		Height: 96,
	}
	testFile2 = webib0.IBFileInfo{
		ID:       "2.jpg",
		Type:     "image",
		Thumb:    testThumb2,
		Original: "original test file 2.jpg",
		Size:     12345,
		Options: map[string]interface{}{
			"width":  1360,
			"height": 1020,
		},
	}
	testThumb3 = webib0.IBThumbInfo{
		ID:     "3.png.jpg",
		Width:  128,
		Height: 128,
	}
	testFile3 = webib0.IBFileInfo{
		ID:       "3.png",
		Type:     "image",
		Thumb:    testThumb3,
		Original: "original test file 3.png",
		Size:     1234567,
		Options: map[string]interface{}{
			"width":  922,
			"height": 922,
		},
	}
	testThumb4 = webib0.IBThumbInfo{
		ID:     "4.opus.jpg",
		Width:  128,
		Height: 128,
	}
	testFile4 = webib0.IBFileInfo{
		ID:       "4.opus",
		Type:     "audio",
		Thumb:    testThumb4,
		Original: "original test file 4.opus",
		Size:     12345678901,
		Options:  map[string]interface{}{},
	}
)
var (
	testPost1 = webib0.IBPostInfo{
		ID:      "0123456789ABCDEF0123456789ABCDEF",
		Subject: "test subject",
		Name:    "Anonymous",
		Trip:    "",
		Date:    1072396800,
		Message: []byte("test OP message"),
		Files: []webib0.IBFileInfo{
			testFile1,
			testFile2,
		},
	}
	testPost2 = webib0.IBPostInfo{
		ID:      "0123456789ABCDEF0123456789ABCDEF",
		Subject: "test subject",
		Name:    "Anonymous",
		Trip:    "",
		Date:    1072396800,
		Message: []byte("test OP message"),
		Files: []webib0.IBFileInfo{
			testFile1,
		},
	}
	testPost3 = webib0.IBPostInfo{
		ID:      "00112233445566770011223344556677",
		Name:    "Anonymous",
		Trip:    "",
		Subject: "",
		Date:    1072396801,
		Message: []byte("test reply message 1"),
		Files:   []webib0.IBFileInfo{},
	}
	testPost4 = webib0.IBPostInfo{
		ID:      "8899AABBCCDDEEFF8899AABBCCDDEEFF",
		Name:    "bob",
		Trip:    "",
		Subject: "",
		Date:    1072396802,
		Message: []byte("test reply message 2"),
		Files: []webib0.IBFileInfo{
			testFile2,
			testFile3,
			testFile4,
		},
	}
)

var (
	testBoardList = webib0.IBBoardList{
		Node: testNodeInfo,
		Boards: []webib0.IBBoardListBoard{
			{"test", "board for testing", []string{"test"}},
			{"testname2", "test description 2", []string{"test", "test2"}},
			{"testname3", "test description 3", []string{"test3", "test4", "test5"}},
			{"testname4", "test description 4", []string{}},
			{"testname5", "test description 5", []string{"test"}},
		},
	}
	testThreadListPage = webib0.IBThreadListPage{
		Node:  testNodeInfo,
		Board: testBoardInfo,
		Threads: []webib0.IBThreadListPageThread{{
			IBCommonThread: webib0.IBCommonThread{
				ID: "0123456789ABCDEF0123456789ABCDEF",
				OP: testPost1,
				Replies: []webib0.IBPostInfo{
					testPost2,
					testPost3,
					testPost4,
				},
			},
			SkippedAttachments: 0,
			SkippedReplies:     0,
		}},
		Avaiable: 2,
	}
	testThread = webib0.IBThreadPage{
		Node:  testNodeInfo,
		Board: testBoardInfo,
		IBCommonThread: webib0.IBCommonThread{
			ID: "0123456789ABCDEF0123456789ABCDEF",
			OP: testPost1,
			Replies: []webib0.IBPostInfo{
				testPost2,
				testPost3,
				testPost4,
			},
		},
	}
	testThreadCatalog = webib0.IBThreadCatalog{
		Node:  testNodeInfo,
		Board: testBoardInfo,
		Threads: []webib0.IBThreadCatalogThread{
			{
				ID:               "0123456789ABCDEF0123456789ABCDEF",
				Thumb:            testThumb1,
				TotalReplies:     0,
				TotalAttachments: 0,
				Subject:          "test1",
				Message:          []byte("test message 1"),
			},
			{
				ID:               "00112233445566770011223344556677",
				Thumb:            testThumb2,
				TotalReplies:     2,
				TotalAttachments: 0,
				Subject:          "test2",
				Message:          []byte(""),
			},
			{
				ID:               "8899AABBCCDDEEFF8899AABBCCDDEEFF",
				Thumb:            testThumb3,
				TotalReplies:     5,
				TotalAttachments: 3,
				Subject:          "",
				Message:          []byte("test message 3"),
			},
		},
	}
)

func (IBProviderDemo) IBGetBoardList(bl *webib0.IBBoardList) (error, int) {
	*bl = testBoardList
	return nil, 0
}

func (IBProviderDemo) IBGetThreadListPage(page *webib0.IBThreadListPage, board string, num uint32) (error, int) {
	if board != "test" {
		return errors.New("board does not exist"), http.StatusNotFound
	}
	if num > 1 {
		return errors.New("page does not exist"), http.StatusNotFound
	}
	*page = testThreadListPage
	page.Number = num
	return nil, 0
}

func (IBProviderDemo) IBGetThreadCatalog(catalog *webib0.IBThreadCatalog, board string) (error, int) {
	if board != "test" {
		return errors.New("board does not exist"), http.StatusNotFound
	}
	*catalog = testThreadCatalog
	return nil, 0
}

func (IBProviderDemo) IBGetThread(thread *webib0.IBThreadPage, board string, threadid string) (error, int) {
	if board != "test" {
		return errors.New("board does not exist"), http.StatusNotFound
	}
	if threadid != testThread.ID {
		return errors.New("thread does not exist"), http.StatusNotFound
	}
	*thread = testThread
	return nil, 0
}
