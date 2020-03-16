// SMTL ファイルのパージング

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

const (
	smtlPkgName = "smtl"
)

// parseSmtlFile は SMTL ファイルをパースし、main 関数の中のステートメントリストを取得する関数。
func parseSmtlFile(smtFilePath string) (stmts []ast.Stmt, err error) {

	// golang の構文としてパースし、ファイルノードを取得
	var fileNode *ast.File
	fset := token.NewFileSet()
	fileNode, err = parser.ParseFile(fset, smtFilePath, nil, 0)
	if err != nil {
		return
	}

	//ast.Print(fset, fileNode)
	//ast.Print(fset, fileNode.Decls)

	// パッケージ名が "smtl" かどうかチェックする
	if fileNode.Name.Name != smtlPkgName {
		err = fmt.Errorf("%s is not supported package", fileNode.Name.Name)
		return
	}

	// ファイルノードのトップレベルの「宣言」の中から main 関数を
	// 見つけ出し、そのステートメントリストを抽出する。
	for _, n := range fileNode.Decls {
		// 関数宣言のうちその名前が "main" のものをみつける
		funcDecl, ok := n.(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == "main" {
			stmts = funcDecl.Body.List
			break
		}
	}
	return
}
