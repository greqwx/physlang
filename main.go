// точка входа: разбор флагов, чтение реестра и задачи, печать решения
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// встроенный реестр нужен, чтобы программа работала без файла рядом
// если рядом с программой лежит formulas.txt, он важнее встроенного
//
//go:embed formulas.txt
var embeddedFormulas string

// main разбирает аргументы и передаёт работу в run
func main() {
	regPath := flag.String("f", "", "файл с формулами (по умолчанию formulas.txt рядом с программой)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "использование: %s [-f формулы.txt] задача.ph\n", filepath.Base(os.Args[0]))
	}
	flag.Parse()
	// без имени файла печатается подсказка по запуску
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *regPath); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

// run решает все пункты «найти» и печатает таблицы по порядку
func run(taskPath, regPath string) error {
	reg, err := loadRegistry(regPath)
	if err != nil {
		return err
	}
	t, err := parseTask(taskPath)
	if err != nil {
		return err
	}

	known := make(map[string]bool, len(t.given))
	for _, q := range t.given {
		known[q.name] = true
	}

	// между вариантами одной задачи остаётся пустая строка
	var out strings.Builder
	first := true
	for _, target := range t.target {
		if known[target] {
			return fmt.Errorf("%s: величина «%s» уже задана в «дано»", taskPath, target)
		}
		vars := selectFormulas(reg, target, known)
		if len(vars) == 0 {
			return noFormulaError(reg, target, known)
		}
		for i, f := range vars {
			sol, err := solve(t, f)
			if err != nil {
				return fmt.Errorf("величина «%s», формула «%s»: %v", target, f.rhs, err)
			}
			if !first {
				out.WriteString("\n")
			}
			first = false
			out.WriteString(renderBlock(t, target, sol, len(vars) > 1, i+1))
		}
	}
	fmt.Print(out.String())
	return nil
}
