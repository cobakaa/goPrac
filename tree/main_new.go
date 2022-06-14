package main

import (
	"fmt"
	"io"
	"os"
)

func removeIndex(slice []os.DirEntry, s int) []os.DirEntry {
	return append(slice[:s], slice[s+1:]...)
}

func removeFiles(files []os.DirEntry) []os.DirEntry {
	var toDel []int
	for i, file := range files {
		if !file.IsDir() {
			toDel = append(toDel, i)
		}
	}
	for j, i := range toDel {
		files = removeIndex(files, i-j)
	}

	return files
}

func fileSizeStr(file os.DirEntry) string {
	res := ""
	if !file.IsDir() {
		fi, _ := file.Info()
		if fi.Size() == 0 {
			res += " (empty)"
		} else {
			res += fmt.Sprintf(" (%db)", fi.Size())
		}
	}
	return res
}

func woFilesTree(path string, depth int, lasts []bool, flag bool) (string, error) {
	res := ""
	files, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("can't read dir %s", path)
	}
	if !flag {
		files = removeFiles(files)
	}
	filesNum := len(files)
	str := ""
	for i, file := range files {
		for _, i := range lasts {
			if !i {
				res += "│\t"
			} else {
				res += "\t"
			}
		}
		if i+1 != filesNum {
			res += "├───" + file.Name() + fileSizeStr(file) + "\n"
			str, _ = woFilesTree(path+"/"+file.Name(), depth+1, append(lasts, false), flag)
			//res += str
		} else {
			res += "└───" + file.Name() + fileSizeStr(file) + "\n"
			str, _ = woFilesTree(path+"/"+file.Name(), depth+1, append(lasts, true), flag)
		}
		res += str
	}
	return res, nil
}

func dirTree(output io.Writer, path string, flag bool) error {
	str := ""
	var err error
	var lasts []bool
	str, err = woFilesTree(path, 0, lasts, flag)

	if err != nil {
		return err
	}
	fmt.Fprint(output, str)
	//fmt.Fprintln(os.Stdout, str)
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
