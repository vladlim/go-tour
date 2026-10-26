package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	command1 = "com1"
	command2 = "com2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("expected command com1 or com2")
		os.Exit(1)
	}

	switch os.Args[1] {
	case command1:
		runCommand1(os.Args[2:])
	case command2:
		runCommand2(os.Args[2:])
	default:
		fmt.Println("unexpected command")
		os.Exit(1)
	}
}

func runCommand1(args []string) {
	f := flag.NewFlagSet(command1, flag.ExitOnError)
	s := f.String("str", "", "строка для работы")
	num := f.Int("num", 1, "количество выполнений")

	err := f.Parse(args)
	if err != nil {
		fmt.Printf("error parsing flags: %s", err.Error())
		os.Exit(1)
	}

	if len(*s) == 0 {
		fmt.Printf("input is empty for %s\n", command1)
		os.Exit(1)
	} else if *num <= 0 {
		fmt.Printf("num is less than zero for %s\n", command1)
		os.Exit(1)
	}

	var builder strings.Builder

	for i := 0; i < *num; i++ {
		builder.WriteString(*s)
	}

	fmt.Println(builder.String())
}

func runCommand2(args []string) {
	f := flag.NewFlagSet(command2, flag.ExitOnError)
	s := f.String("str", "", "строка для работы")
	likeError := f.Bool("likeError", false, "вывести строку как ошибку?")

	err := f.Parse(args)
	if err != nil {
		fmt.Printf("error parsing flags: %s", err.Error())
		os.Exit(1)
	}

	if len(*s) == 0 {
		fmt.Printf("input is empty for %s\n", command1)
		os.Exit(1)
	}

	switch *likeError {
	case true:
		fmt.Println(fmt.Errorf("error: %w", errors.New(*s)))
	default:
		fmt.Printf("not error: %s\n", *s)
	}
}
