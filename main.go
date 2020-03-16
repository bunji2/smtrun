// SMTファイルに記述された制約関係を解決するプログラム

package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/mitchellh/go-z3"
)

const (
	cmdFmt = "Usage: %s file.smt\n"
)

func main() {
	os.Exit(run())
}

func run() int {
	// 引数チェック
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, cmdFmt, os.Args[0])
		return 1
	}

	fmt.Println("smtrun", VERSION)

	smtlFilePath := os.Args[1]

	// コンテクストオブジェクトの作成
	config := z3.NewConfig()
	ctx := z3.NewContext(config)
	config.Close()
	defer ctx.Close()

	// 変数テーブル初期化
	varTab := map[string]*z3.AST{}

	// SMTファイルの処理
	solver, err := processSmtlFile(ctx, varTab, smtlFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	defer solver.Close()

	// 解決可能かどうかをチェック
	if v := solver.Check(); v != z3.True {
		fmt.Println("Unsolveable")
		return 3
	}

	// 結果となるモデルを取得
	m := solver.Model()
	assignments := m.Assignments()
	m.Close()

	// 変数名を取得
	var names []string
	for name := range varTab {
		names = append(names, name)
	}
	sort.Strings(names)

	// 制約関係を満たす変数の値を表示
	for _, name := range names {
		fmt.Printf("%s = %s\n", name, assignments[name])
	}

	return 0
}
