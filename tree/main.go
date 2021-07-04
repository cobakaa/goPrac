package main

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type FileInfo struct {
	Path  string
	Size  int64
	IsDir bool
}

type ByPath []FileInfo

func (a ByPath) Len() int {
	return len(a)
}

func (a ByPath) Less(i, j int) bool {
	return a[i].Path < a[j].Path
}

func (a ByPath) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func genUpperDir(path string) (res string) {
	curPath := strings.Split(path, "/")

	for i := 0; i < len(curPath)-1; i++ {
		res += curPath[i]

		if i != len(curPath)-2 {
			res += "/"
		}
	}
	return
}

func dirTree(out io.Writer, path string, printFiles bool) error {

	var paths []FileInfo

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		file := FileInfo{p, info.Size(), info.IsDir()}
		if !info.IsDir() && !printFiles {
			return nil
		}
		paths = append(paths, file)
		return nil
	})

	paths = paths[1:]
	sort.Sort(ByPath(paths))

	curLvl := 0
	pathMap := make(map[string][]string)
	for _, f := range paths {

		upperPath := genUpperDir(f.Path)
		pathMap[upperPath] = append(pathMap[upperPath], f.Path)

	}

	curLvl = 0

	for _, f := range paths {
		curPath := strings.Split(f.Path, "/")

		curLen := len(curPath)

		curLvl = curLen - 1

		var res string

		for i := range curPath {

			var str string
			var upperPath string
			for j := 0; j <= i; j++ {
				str += curPath[j]
				if j < i {
					upperPath += curPath[j]
					if j != i-1 {
						upperPath += "/"
					}
				}

				if i != j {
					str += "/"
				}

			}
			if i < curLvl {

				if val, b := pathMap[upperPath]; b && val[len(val)-1] == str {
					res += "\t"
				} else if b {
					res += "│\t"
				}
			}
		}

		var upperPath = genUpperDir(f.Path)

		if val, b := pathMap[upperPath]; b && val[len(val)-1] == f.Path {
			res += "└───"
		} else {
			res += "├───"
		}

		res += curPath[curLvl]
		if !f.IsDir {
			if f.Size == 0 {
				res += " (empty)"
			} else {
				res += " (" + strconv.FormatInt(f.Size, 10) + "b)"
			}
		}
		res += "\n"

		io.WriteString(out, res)

	}

	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
