package demoib

import (
	"../webib0"
	"errors"
	"net/http"
)

type IBProviderDemo struct{}

var _ webib0.IBProvider = (*IBProviderDemo)(nil)

func (IBProviderDemo) IBGetBoardList(bl *webib0.IBBoardList) (error, int) {
	*bl = webib0.IBBoardList{Boards: []webib0.IBBoardListBoard{
		{"test", "board for testing", []string{"test"}},
		{"testname2", "test description 2", []string{"test", "test2"}},
		{"testname3", "test description 3", []string{"test3", "test4", "test5"}},
		{"testname4", "test description 4", []string{}},
		{"testname5", "test description 5", []string{"test"}},
	}}
	return nil, 0
}

var testBoardInfo = webib0.IBBoardInfo{
	"test",
	"board for testing",
	"nothing of value visible there",
}
var (
	testThumb1 = webib0.IBThumbInfo{
		ID:     "1.png.jpg",
		Width:  128,
		Height: 128,
	}
	testThumb2 = webib0.IBThumbInfo{
		ID:     "2.jpg.jpg",
		Width:  128,
		Height: 96,
	}
	testThumb3 = webib0.IBThumbInfo{
		ID:     "3.png.jpg",
		Width:  128,
		Height: 128,
	}
)
var (
	testPost1 = webib0.IBPostInfo{
		ID:      "0123456789ABCDEF0123456789ABCDEF",
		Subject: "test subject",
		Name:    "",
		Trip:    "",
		Date:    1072396800,
		Message: "test OP message",
		Files: []webib0.IBFileInfo{
			{
				ID:       "1.png",
				Type:     "image",
				Thumb:    testThumb1,
				Original: "original test file.png",
				Options: map[string]interface{}{
					"width":  512,
					"height": 512,
				},
			},
		},
	}
	testPost2 = webib0.IBPostInfo{
		ID:      "00112233445566770011223344556677",
		Name:    "",
		Trip:    "",
		Subject: "",
		Date:    1072396801,
		Message: "test reply message 1",
		Files:   []webib0.IBFileInfo{},
	}
	testPost3 = webib0.IBPostInfo{
		ID:      "8899AABBCCDDEEFF8899AABBCCDDEEFF",
		Name:    "bob",
		Trip:    "",
		Subject: "",
		Date:    1072396802,
		Message: "test reply message 2",
		Files: []webib0.IBFileInfo{
			{
				ID:       "2.jpg",
				Type:     "image",
				Thumb:    testThumb2,
				Original: "original test file 2.jpg",
				Options: map[string]interface{}{
					"width":  512,
					"height": 256,
				},
			},
			{
				ID:       "3.opus",
				Type:     "audio",
				Thumb:    testThumb3,
				Original: "original test file 3.opus",
				Options:  map[string]interface{}{},
			},
		},
	}
)

func (IBProviderDemo) IBGetThreadListPage(page *webib0.IBThreadListPage, board string, num uint32) (error, int) {
	if board != "test" {
		return errors.New("board does not exist"), http.StatusNotFound
	}
	if num > 1 {
		return errors.New("page does not exist"), http.StatusNotFound
	}
	*page = webib0.IBThreadListPage{
		Board: testBoardInfo,
		Threads: []webib0.IBThreadListPageThread{{
			ID:                 "0123456789ABCDEF0123456789ABCDEF",
			SkippedAttachments: 0,
			SkippedReplies:     0,
			OP:                 testPost1,
			Replies: []webib0.IBPostInfo{
				testPost2,
				testPost3,
			},
		}},
		Number:   num,
		Avaiable: 2,
	}
	return nil, 0
}

func (IBProviderDemo) IBGetThreadCatalog(catalog *webib0.IBThreadCatalog, board string) (error, int) {
	if board != "test" {
		return errors.New("board does not exist"), http.StatusNotFound
	}
	*catalog = webib0.IBThreadCatalog{
		Board: testBoardInfo,
		Threads: []webib0.IBThreadCatalogThread{
			{
				ID:               "0123456789ABCDEF0123456789ABCDEF",
				Thumb:            testThumb1,
				TotalReplies:     0,
				TotalAttachments: 0,
				Subject:          "test1",
				Message:          "test message 1",
			},
			{
				ID:               "00112233445566770011223344556677",
				Thumb:            testThumb2,
				TotalReplies:     2,
				TotalAttachments: 0,
				Subject:          "test2",
				Message:          "",
			},
			{
				ID:               "8899AABBCCDDEEFF8899AABBCCDDEEFF",
				Thumb:            testThumb3,
				TotalReplies:     5,
				TotalAttachments: 3,
				Subject:          "",
				Message:          "test message 3",
			},
		},
	}
	return nil, 0
}

func (IBProviderDemo) IBGetThread(thread *webib0.IBThreadPage, board string, threadid string) (error, int) {
	if board != "test" {
		return errors.New("board does not exist"), http.StatusNotFound
	}
	if threadid != "0123456789ABCDEF0123456789ABCDEF" {
		return errors.New("thread does not exist"), http.StatusNotFound
	}
	*thread = webib0.IBThreadPage{
		Board: testBoardInfo,
		ID:    "0123456789ABCDEF0123456789ABCDEF",
		OP:    testPost1,
		Replies: []webib0.IBPostInfo{
			testPost2,
			testPost3,
		},
	}
	return nil, 0
}
