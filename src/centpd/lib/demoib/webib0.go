package demoib

import (
	"errors"
	"net/http"

	"centpd/lib/webib0"
)

var bible1 = []byte(`In the beginning God created the heaven and the earth. And the earth was without form, and void; and darkness was upon the face of the deep. And the Spirit of God moved upon the face of the waters.
And God said, Let there be light: and there was light.
And God saw the light, that it was good: and God divided the light from the darkness.
And God called the light Day, and the darkness he called Night. And the evening and the morning were the first day.
And God said, Let there be a firmament in the midst of the waters, and let it divide the waters from the waters.
And God made the firmament, and divided the waters which were under the firmament from the waters which were above the firmament: and it was so.
And God called the firmament Heaven. And the evening and the morning were the second day.
And God said, Let the waters under the heaven be gathered together unto one place, and let the dry land appear: and it was so.
And God called the dry land Earth; and the gathering together of the waters called he Seas: and God saw that it was good.
And God said, Let the earth bring forth grass, the herb yielding seed, and the fruit tree yielding fruit after his kind, whose seed is in itself, upon the earth: and it was so.
And the earth brought forth grass, and herb yielding seed after his kind, and the tree yielding fruit, whose seed was in itself, after his kind: and God saw that it was good.
And the evening and the morning were the third day.
And God said, Let there be lights in the firmament of the heaven to divide the day from the night; and let them be for signs, and for seasons, and for days, and years:
And let them be for lights in the firmament of the heaven to give light upon the earth: and it was so.
And God made two great lights; the greater light to rule the day, and the lesser light to rule the night: he made the stars also.
And God set them in the firmament of the heaven to give light upon the earth,
And to rule over the day and over the night, and to divide the light from the darkness: and God saw that it was good.
And the evening and the morning were the fourth day.
And God said, Let the waters bring forth abundantly the moving creature that hath life, and fowl that may fly above the earth in the open firmament of heaven.
And God created great whales, and every living creature that moveth, which the waters brought forth abundantly, after their kind, and every winged fowl after his kind: and God saw that it was good.
And God blessed them, saying, Be fruitful, and multiply, and fill the waters in the seas, and let fowl multiply in the earth.
And the evening and the morning were the fifth day.
And God said, Let the earth bring forth the living creature after his kind, cattle, and creeping thing, and beast of the earth after his kind: and it was so.
And God made the beast of the earth after his kind, and cattle after their kind, and every thing that creepeth upon the earth after his kind: and God saw that it was good.
And God said, Let us make man in our image, after our likeness: and let them have dominion over the fish of the sea, and over the fowl of the air, and over the cattle, and over all the earth, and over every creeping thing that creepeth upon the earth.
So God created man in his own image, in the image of God created he him; male and female created he them.
And God blessed them, and God said unto them, Be fruitful, and multiply, and replenish the earth, and subdue it: and have dominion over the fish of the sea, and over the fowl of the air, and over every living thing that moveth upon the earth.
And God said, Behold, I have given you every herb bearing seed, which is upon the face of all the earth, and every tree, in the which is the fruit of a tree yielding seed; to you it shall be for meat.
And to every beast of the earth, and to every fowl of the air, and to every thing that creepeth upon the earth, wherein there is life, I have given every green herb for meat: and it was so.
And God saw every thing that he had made, and, behold, it was very good. And the evening and the morning were the sixth day.`)

type IBProviderDemo struct{}

var _ webib0.IBProvider = (*IBProviderDemo)(nil)

var testBoardInfo = webib0.IBBoardInfo{
	Name:        "test",
	Description: "board for testing",
	Info:        "nothing of value visible there",
}
var (
	testThumb1 = webib0.IBThumbInfo{
		ID: "1.png.jpg",
		IBThumbAttributes: webib0.IBThumbAttributes{
			Width:  128,
			Height: 128,
		},
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
		ID: "2.jpg.jpg",
		IBThumbAttributes: webib0.IBThumbAttributes{Width: 128,
			Height: 96,
		},
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
		ID: "3.png.jpg",
		IBThumbAttributes: webib0.IBThumbAttributes{
			Width:  128,
			Height: 128,
		},
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
		ID: "4.opus.jpg",
		IBThumbAttributes: webib0.IBThumbAttributes{
			Width:  128,
			Height: 128,
		},
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
		Message: bible1,
		Files: []webib0.IBFileInfo{
			testFile1,
			//testFile2,
		},
	}
	testPost2 = webib0.IBPostInfo{
		ID:      "0123456789ABCDEF0123456789ABCDEF",
		Subject: "test subject",
		Name:    "Anonymous",
		Trip:    "",
		Date:    1072396800,
		Message: []byte("test reply msg 0"),
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
	testPost5 = webib0.IBPostInfo{
		ID:      "8899AABBCCDDEEFF8899AABBCCDDEEFF",
		Name:    "Anonymous",
		Trip:    "",
		Subject: "",
		Date:    1072396803,
		Message: []byte(">testing greentext\nnon-greentext\n>greentext again"),
		Files: []webib0.IBFileInfo{
			testFile2,
			testFile3,
			testFile4,
		},
	}
)

var (
	testBoardList = webib0.IBBoardList{
		Boards: []webib0.IBBoardListBoard{
			{"test", "board for testing", []string{"test"}, 0, 0},
			{"testname2", "test description 2", []string{"test", "test2"}, 0, 0},
			{"testname3", "test description 3", []string{"test3", "test4", "test5"}, 0, 0},
			{"testname4", "test description 4", []string{}, 0, 0},
			{"testname5", "test description 5", []string{"test"}, 0, 0},
		},
	}
	testThreadListPage = webib0.IBThreadListPage{
		Board: testBoardInfo,
		Threads: []webib0.IBThreadListPageThread{{
			IBCommonThread: webib0.IBCommonThread{
				ID: "0123456789ABCDEF0123456789ABCDEF",
				OP: testPost1,
				Replies: []webib0.IBPostInfo{
					testPost2,
					testPost3,
					testPost4,
					testPost5,
				},
			},
			SkippedFiles:   0,
			SkippedReplies: 0,
		}},
		Available: 2,
	}
	testThread = webib0.IBThreadPage{
		Board: testBoardInfo,
		IBCommonThread: webib0.IBCommonThread{
			ID: "0123456789ABCDEF0123456789ABCDEF",
			OP: testPost1,
			Replies: []webib0.IBPostInfo{
				testPost2,
				testPost3,
				testPost4,
				testPost5,
			},
		},
	}
	testThreadCatalog = webib0.IBThreadCatalog{
		Board: testBoardInfo,
		Threads: []webib0.IBThreadCatalogThread{
			{
				ID:           "0123456789ABCDEF0123456789ABCDEF",
				Thumb:        testThumb1,
				TotalReplies: 0,
				TotalFiles:   0,
				Subject:      "test1",
				Message:      []byte("test message 1"),
			},
			{
				ID:           "00112233445566770011223344556677",
				Thumb:        testThumb2,
				TotalReplies: 2,
				TotalFiles:   0,
				Subject:      "test2",
				Message:      []byte(""),
			},
			{
				ID:           "8899AABBCCDDEEFF8899AABBCCDDEEFF",
				Thumb:        testThumb3,
				TotalReplies: 5,
				TotalFiles:   3,
				Subject:      "",
				Message:      []byte("test message 3"),
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
