package main

import (
	"fmt"
	"github.com/go-faker/faker/v4"
	"math"
	"math/rand"
	"time"
)

type FakeFile struct {
	IsDir      bool
	isHidden   bool
	FileName   string
	FileSize   int
	ModTime    time.Time
	Perm       string
	Owner      string
	OwnerGroup string
}

var fileOwners = []string{
	"huahuo",
	"fumo",
	"root",
}
var fileExtArrays = []string{
	".txt", ".doc", ".docx", ".pdf", ".xls", ".xlsx",
	".ppt", ".pptx", ".jpg", ".jpeg", ".png", ".gif",
	".bmp", ".tiff", ".csv", ".zip", ".rar", ".tar",
	".gz", ".7z", ".mp3", ".wav", ".mp4", ".avi",
	".mov", ".mkv", ".flv", ".html", ".css", ".js",
	".json", ".xml", ".sql", ".py", ".java", ".c",
	".cpp", ".h", ".hpp", ".go", ".php", ".rb",
	".swift", ".kt", ".ts", ".tsx", ".vue", ".md",
	".log", ".ini", ".conf", ".bat", ".sh", ".ps1",
}

var filePermissions = []string{
	"rwx",
	"rw-",
	"r-x",
	"r--",
	"-wx",
	"-w-",
	"--x",
	"---",
}

func getRandomValue(arr []string) string {
	randomIndex := rand.Intn(len(arr))
	randomValue := arr[randomIndex]
	return randomValue
}

func fakeFileList(count int) []FakeFile {
	files := make([]FakeFile, count)
	dirIndex := rand.Intn(count)
	hiddenIndex := rand.Intn(count-dirIndex) % 20
	for i := 0; i < count; i++ {
		owner := getRandomValue(fileOwners)
		files[i] = FakeFile{
			IsDir:      i < dirIndex,
			FileName:   faker.Word(),
			FileSize:   rand.Intn(2 << 16),
			ModTime:    time.Now().Add(-time.Duration(rand.Intn(math.MaxInt >> 4))),
			Perm:       "",
			Owner:      owner,
			OwnerGroup: owner,
		}

		for j := 0; j < 3; j += 1 {
			files[i].Perm += getRandomValue(filePermissions)
		}

		if i >= dirIndex {
			files[i].isHidden = (i - dirIndex) < hiddenIndex
		}
		if !files[i].IsDir {
			if files[i].isHidden {
				files[i].FileName = fmt.Sprintf(".%s", files[i].FileName)
			} else {
				ext := getRandomValue(fileExtArrays)
				files[i].FileName += ext
			}
		}
	}
	return files
}
