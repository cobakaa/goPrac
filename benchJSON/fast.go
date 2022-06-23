package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	// "log"
)

const filePath string = "./data/users.txt"

type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
}

func FastSearch(out io.Writer) {

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	fmt.Fprintln(out, "found users:")

	i := 0
	browsers := map[string]bool{}

	scanner := bufio.NewScanner(file)
	user := User{}
	for scanner.Scan() {
		if err = user.UnmarshalJSON(scanner.Bytes()); err != nil {
			panic(err)
		}

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browsers {
			if strings.Contains(browser, "Android") {
				isAndroid = true
			} else if strings.Contains(browser, "MSIE") {
				isMSIE = true
			} else {
				continue
			}

			browsers[browser] = true
		}

		if isAndroid && isMSIE {
			email := strings.Replace(user.Email, "@", " [at] ", -1)
			fmt.Fprintf(out, "[%d] %s <%s>\n", i, user.Name, email)
		}

		i++
	}

	fmt.Fprintf(out, "\nTotal unique browsers %d\n", len(browsers))
}
