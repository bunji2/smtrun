// SMTL ファイルを読み込み、使用する変数と制約関係を登録する。
// ここでは go/ast を使って SMTL ファイルから go-AST を取得し、
// 制約関係に対応する go-AST から go-z3 の AST を構築していく。

package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/mitchellh/go-z3"
)

// processSmtlFile は SMTL ファイルを処理する関数。
func processSmtlFile(ctx *z3.Context, varTab map[string]*z3.AST, smtFilePath string) (s *z3.Solver, err error) {
	// SMT ファイルのパース。main 関数の中のステートメントリストを取得。
	var stmts []ast.Stmt
	stmts, err = parseSmtlFile(smtFilePath)
	if err != nil {
		return
	}

	s = ctx.NewSolver()

	// 各ステートメントを処理
	for _, stmt := range stmts {
		err = processStmt(ctx, s, varTab, stmt)
		if err != nil {
			break
		}
	}

	return
}

// processStmt はステートメントを処理する関数。
func processStmt(ctx *z3.Context, s *z3.Solver, varTab map[string]*z3.AST, stmt ast.Stmt) (err error) {
	switch stmt.(type) {
	case *ast.DeclStmt: // 宣言に関するステートメント
		//fmt.Println("DeclStmt!")
		err = processDeclStmt(ctx, s, varTab, stmt.(*ast.DeclStmt))
	case *ast.ExprStmt: // 式に関するステートメント
		//fmt.Println("ExprStmt!")
		err = processExprStmt(ctx, s, varTab, stmt.(*ast.ExprStmt))
	default:
		// その他のステートメントはエラー
		err = fmt.Errorf("not supported Stmt")
	}
	return
}

// processDeclStmt は宣言ステートメントを処理する関数。
func processDeclStmt(ctx *z3.Context, s *z3.Solver, varTab map[string]*z3.AST, decl *ast.DeclStmt) (err error) {
	//fmt.Println("DeclStmt!")

	// 変数宣言 (var x TYPE) ならば変数を登録する
	gd, ok := decl.Decl.(*ast.GenDecl)
	if ok && gd.Tok == token.VAR {
		err = processVarSpec(ctx, varTab, gd.Specs[0].(*ast.ValueSpec))
	} else {
		err = fmt.Errorf("not supported Tok of DeclStmt")
	}
	return
}

// processVarSpec は変数宣言を処理する関数。
// 変数は varTab に登録される。
func processVarSpec(ctx *z3.Context, varTab map[string]*z3.AST, vs *ast.ValueSpec) (err error) {

	// 変数の型の確認
	var sort *z3.Sort

	switch vs.Type.(type) {
	case *ast.Ident:
		id := vs.Type.(*ast.Ident)
		switch id.Name {
		case "int":
			sort = ctx.IntSort()
		case "bool":
			sort = ctx.BoolSort()

			// 対応する型を増やす場合はここに挿入

		default:
			// 非対応の型
			err = fmt.Errorf("type %s is not supported", id.Name)
		}

	case *ast.ArrayType:
		err = fmt.Errorf("ArrayType of VarSpec is not supported")

	default:
		err = fmt.Errorf("not supported Type of ValueSpec")
	}

	if err != nil {
		return
	}

	// 各変数の処理
	for _, name := range vs.Names {
		// 変数名の重複は禁止
		if _, ok := varTab[name.Name]; ok {
			err = fmt.Errorf("var %s is already declared", name.Name)
			break
		}
		varTab[name.Name] = ctx.Const(ctx.Symbol(name.Name), sort)
	}

	return
}

// processExprStmt は式のステートメントを処理する関数。
func processExprStmt(ctx *z3.Context, s *z3.Solver, varTab map[string]*z3.AST, exprStmt *ast.ExprStmt) (err error) {
	// main 関数直下の assert 関数のみを処理する。

	// 関数呼び出しかどうかをチェック
	ce, ok := exprStmt.X.(*ast.CallExpr)
	if ok {
		// identifier (args) の形の関数呼び出しか
		fun, ok := ce.Fun.(*ast.Ident)
		if ok && fun.Name == "assert" {
			var x *z3.AST
			args := ce.Args
			if len(args) != 1 {
				// assert 関数の引数が１以外（０もしくは２以上）の場合はエラー
				err = fmt.Errorf("assert must have single argument")
				return
			}
			// assert 関数の第一引数の z3.AST を取得する。
			x, err = processExpr(ctx, varTab, args[0])
			if err != nil {
				return
			}

			// z3 に登録する。
			s.Assert(x)
		} else {
			// 他の形式の関数呼び出しはサポート外
			err = fmt.Errorf("not supported Fun of CallExpr")
		}
	} else {
		// 関数呼び出し以外の式ステートメントはサポート外
		err = fmt.Errorf("not supported X of ExprStmt")
	}
	return
}

// processExpr は入力された式に応じた z3.AST を作成する関数
func processExpr(ctx *z3.Context, varTab map[string]*z3.AST, expr ast.Expr) (r *z3.AST, err error) {
	switch expr.(type) {
	case *ast.Ident:
		r, err = processIdent(ctx, varTab, expr.(*ast.Ident))

	case *ast.BasicLit:
		r, err = processBasicLit(ctx, varTab, expr.(*ast.BasicLit))

	case *ast.BinaryExpr:
		r, err = processBinaryExpr(ctx, varTab, expr.(*ast.BinaryExpr))

	case *ast.UnaryExpr:
		r, err = processUnaryExpr(ctx, varTab, expr.(*ast.UnaryExpr))

	case *ast.CallExpr:
		r, err = processCallExpr(ctx, varTab, expr.(*ast.CallExpr))

	case *ast.ParenExpr:
		pe := expr.(*ast.ParenExpr)
		r, err = processExpr(ctx, varTab, pe.X)

	default:
		err = fmt.Errorf("not supported Expr")
	}
	return
}

func processIdent(ctx *z3.Context, varTab map[string]*z3.AST, ident *ast.Ident) (r *z3.AST, err error) {
	switch ident.Name {
	case "true":
		r = ctx.True()
	case "false":
		r = ctx.False()
	default:
		if varTab[ident.Name] != nil {
			r = varTab[ident.Name]
		} else {
			err = fmt.Errorf("%s is unknown variable", ident.Name)
		}
	}
	return
}

func processBasicLit(ctx *z3.Context, varTab map[string]*z3.AST, basicLit *ast.BasicLit) (r *z3.AST, err error) {
	if basicLit.Kind == token.INT {
		intVal, err := strconv.Atoi(basicLit.Value)
		if err == nil {
			r = ctx.Int(intVal, ctx.IntSort())
		}
	}
	return
}

// processBinaryExpr は二項演算式を処理し、z3 の AST を作成する関数
func processBinaryExpr(ctx *z3.Context, varTab map[string]*z3.AST, be *ast.BinaryExpr) (r *z3.AST, err error) {
	var x, y *z3.AST
	x, err = processExpr(ctx, varTab, be.X)
	if err == nil {
		y, err = processExpr(ctx, varTab, be.Y)
		if err == nil {
			r, err = processBOP(be.Op, x, y)
		}
	}
	return
}

// processBOP は二項演算子を処理し、z3 の AST を作成する関数
func processBOP(op token.Token, x, y *z3.AST) (r *z3.AST, err error) {
	switch op {
	case token.ADD: // +
		r = x.Add(y)
	case token.SUB: // -
		r = x.Sub(y)
	case token.MUL: // *
		r = x.Mul(y)
	case token.LAND: // &&
		r = x.And(y)
	case token.LOR: // ||
		r = x.Or(y)
	case token.EQL: // ==
		r = x.Eq(y)
	case token.LSS: // <
		r = x.Lt(y)
	case token.GTR: // >
		r = x.Gt(y)
	case token.NEQ: // !=
		r = x.Eq(y).Not()
	case token.LEQ: // <=
		r = x.Le(y)
	case token.GEQ: // >=
		r = x.Ge(y)
	default:
		err = fmt.Errorf("not supported bop")
	}
	return
}

// processUnaryExpr は単項演算式を処理し、z3 の AST を作成する関数
func processUnaryExpr(ctx *z3.Context, varTab map[string]*z3.AST, ue *ast.UnaryExpr) (r *z3.AST, err error) {
	var x *z3.AST
	x, err = processExpr(ctx, varTab, ue.X)
	if err == nil {
		switch ue.Op {
		case token.NOT:
			r = x.Not()
		default:
			err = fmt.Errorf("not supported bop")
		}
	}
	return
}

func processCallExpr(ctx *z3.Context, varTab map[string]*z3.AST, ce *ast.CallExpr) (r *z3.AST, err error) {
	var args []*z3.AST
	var e *z3.AST
	for _, arg := range ce.Args {
		e, err = processExpr(ctx, varTab, arg)
		if err != nil {
			break
		}
		args = append(args, e)
	}
	if len(args) == 0 {
		err = fmt.Errorf("too few argument of CallExpr")
	}
	switch ce.Fun.(type) {
	case *ast.Ident:
		ident := ce.Fun.(*ast.Ident)
		if ident.Name == "distinct" {
			if len(args) > 1 {
				r = args[0].Distinct(args[1:]...)
			} else {
				err = fmt.Errorf("distinct must have 2 arguments at least")
			}
		} else {
			err = fmt.Errorf("not supported Name of Indent")
		}
	case *ast.SelectorExpr:
		se := ce.Fun.(*ast.SelectorExpr)
		var e *z3.AST
		e, err = processExpr(ctx, varTab, se.X)
		if err != nil {
			break
		}
		switch se.Sel.Name {
		case "implies":
			if len(args) == 1 {
				r = e.Implies(args[0])
			} else {
				err = fmt.Errorf("imples must have signle argument")
			}
		case "iff":
			if len(args) == 1 {
				r = e.Iff(args[0])
			} else {
				err = fmt.Errorf("iff must have single argument")
			}
		default:
			err = fmt.Errorf("not supported Sel.Name of SelectorExpr")
		}
	default:
		err = fmt.Errorf("not supported Fun of CallExpr")
	}
	return
}

/*
func isIdent(expr ast.Expr) (name string, ok bool) {
	ident, ok := expr.(*ast.Ident)
	if ok {
		name = ident.Name
	}
	return
}

func isIntVal(expr ast.Expr) (intVal int, ok bool) {
	basicLit, ok := expr.(*ast.BasicLit)
	if ok && basicLit.Kind == token.INT {
		var err error
		intVal, err = strconv.Atoi(basicLit.Value)
		if err != nil {
			ok = false
		}
	}
	return

}
*/

/*
func isBoolVal(expr ast.Expr) (boolVal bool, ok bool) {
	basicLit, ok := expr.(*ast.BasicLit)
	if ok && basicLit.Kind == token.INT {
		intVal, ok = strconv.Atoi(basicLit.Value)
	}
	return

}
*/

/*
     0  *ast.DeclStmt {
     1  .  Decl: *ast.GenDecl {
     2  .  .  TokPos: ./foo.z3:4:2
     3  .  .  Tok: var
     4  .  .  Lparen: -
     5  .  .  Specs: []ast.Spec (len = 1) {
     6  .  .  .  0: *ast.ValueSpec {
     7  .  .  .  .  Names: []*ast.Ident (len = 1) {
     8  .  .  .  .  .  0: *ast.Ident {
     9  .  .  .  .  .  .  NamePos: ./foo.z3:4:6
    10  .  .  .  .  .  .  Name: "x"
    11  .  .  .  .  .  .  Obj: *ast.Object {
    12  .  .  .  .  .  .  .  Kind: var
    13  .  .  .  .  .  .  .  Name: "x"
    14  .  .  .  .  .  .  .  Decl: *(obj @ 6)
    15  .  .  .  .  .  .  .  Data: 0
    16  .  .  .  .  .  .  }
    17  .  .  .  .  .  }
    18  .  .  .  .  }
    19  .  .  .  .  Type: *ast.Ident {
    20  .  .  .  .  .  NamePos: ./foo.z3:4:8
    21  .  .  .  .  .  Name: "int"
    22  .  .  .  .  }
    23  .  .  .  }
    24  .  .  }
    25  .  .  Rparen: -
    26  .  }
    27  }
DeclStmt!
x int
     0  *ast.DeclStmt {
     1  .  Decl: *ast.GenDecl {
     2  .  .  TokPos: ./foo.z3:5:2
     3  .  .  Tok: var
     4  .  .  Lparen: -
     5  .  .  Specs: []ast.Spec (len = 1) {
     6  .  .  .  0: *ast.ValueSpec {
     7  .  .  .  .  Names: []*ast.Ident (len = 1) {
     8  .  .  .  .  .  0: *ast.Ident {
     9  .  .  .  .  .  .  NamePos: ./foo.z3:5:6
    10  .  .  .  .  .  .  Name: "y"
    11  .  .  .  .  .  .  Obj: *ast.Object {
    12  .  .  .  .  .  .  .  Kind: var
    13  .  .  .  .  .  .  .  Name: "y"
    14  .  .  .  .  .  .  .  Decl: *(obj @ 6)
    15  .  .  .  .  .  .  .  Data: 0
    16  .  .  .  .  .  .  }
    17  .  .  .  .  .  }
    18  .  .  .  .  }
    19  .  .  .  .  Type: *ast.Ident {
    20  .  .  .  .  .  NamePos: ./foo.z3:5:8
    21  .  .  .  .  .  Name: "int"
    22  .  .  .  .  }
    23  .  .  .  }
    24  .  .  }
    25  .  .  Rparen: -
    26  .  }
    27  }
DeclStmt!
y int
     0  *ast.DeclStmt {
     1  .  Decl: *ast.GenDecl {
     2  .  .  TokPos: ./foo.z3:7:2
     3  .  .  Tok: var
     4  .  .  Lparen: -
     5  .  .  Specs: []ast.Spec (len = 1) {
     6  .  .  .  0: *ast.ValueSpec {
     7  .  .  .  .  Names: []*ast.Ident (len = 1) {
     8  .  .  .  .  .  0: *ast.Ident {
     9  .  .  .  .  .  .  NamePos: ./foo.z3:7:6
    10  .  .  .  .  .  .  Name: "z"
    11  .  .  .  .  .  .  Obj: *ast.Object {
    12  .  .  .  .  .  .  .  Kind: var
    13  .  .  .  .  .  .  .  Name: "z"
    14  .  .  .  .  .  .  .  Decl: *(obj @ 6)
    15  .  .  .  .  .  .  .  Data: 0
    16  .  .  .  .  .  .  }
    17  .  .  .  .  .  }
    18  .  .  .  .  }
    19  .  .  .  .  Type: *ast.Ident {
    20  .  .  .  .  .  NamePos: ./foo.z3:7:8
    21  .  .  .  .  .  Name: "bool"
    22  .  .  .  .  }
    23  .  .  .  }
    24  .  .  }
    25  .  .  Rparen: -
    26  .  }
    27  }
DeclStmt!
z bool
     0  *ast.ExprStmt {
     1  .  X: *ast.CallExpr {
     2  .  .  Fun: *ast.Ident {
     3  .  .  .  NamePos: ./foo.z3:9:2
     4  .  .  .  Name: "assert"
     5  .  .  }
     6  .  .  Lparen: ./foo.z3:9:8
     7  .  .  Args: []ast.Expr (len = 1) {
     8  .  .  .  0: *ast.BinaryExpr {
     9  .  .  .  .  X: *ast.BinaryExpr {
    10  .  .  .  .  .  X: *ast.Ident {
    11  .  .  .  .  .  .  NamePos: ./foo.z3:9:9
    12  .  .  .  .  .  .  Name: "x"
    13  .  .  .  .  .  .  Obj: *ast.Object {
    14  .  .  .  .  .  .  .  Kind: var
    15  .  .  .  .  .  .  .  Name: "x"
    16  .  .  .  .  .  .  .  Decl: *ast.ValueSpec {
    17  .  .  .  .  .  .  .  .  Names: []*ast.Ident (len = 1) {
    18  .  .  .  .  .  .  .  .  .  0: *ast.Ident {
    19  .  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:4:6
    20  .  .  .  .  .  .  .  .  .  .  Name: "x"
    21  .  .  .  .  .  .  .  .  .  .  Obj: *(obj @ 13)
    22  .  .  .  .  .  .  .  .  .  }
    23  .  .  .  .  .  .  .  .  }
    24  .  .  .  .  .  .  .  .  Type: *ast.Ident {
    25  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:4:8
    26  .  .  .  .  .  .  .  .  .  Name: "int"
    27  .  .  .  .  .  .  .  .  }
    28  .  .  .  .  .  .  .  }
    29  .  .  .  .  .  .  .  Data: 0
    30  .  .  .  .  .  .  }
    31  .  .  .  .  .  }
    32  .  .  .  .  .  OpPos: ./foo.z3:9:10
    33  .  .  .  .  .  Op: +
    34  .  .  .  .  .  Y: *ast.Ident {
    35  .  .  .  .  .  .  NamePos: ./foo.z3:9:11
    36  .  .  .  .  .  .  Name: "y"
    37  .  .  .  .  .  .  Obj: *ast.Object {
    38  .  .  .  .  .  .  .  Kind: var
    39  .  .  .  .  .  .  .  Name: "y"
    40  .  .  .  .  .  .  .  Decl: *ast.ValueSpec {
    41  .  .  .  .  .  .  .  .  Names: []*ast.Ident (len = 1) {
    42  .  .  .  .  .  .  .  .  .  0: *ast.Ident {
    43  .  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:5:6
    44  .  .  .  .  .  .  .  .  .  .  Name: "y"
    45  .  .  .  .  .  .  .  .  .  .  Obj: *(obj @ 37)
    46  .  .  .  .  .  .  .  .  .  }
    47  .  .  .  .  .  .  .  .  }
    48  .  .  .  .  .  .  .  .  Type: *ast.Ident {
    49  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:5:8
    50  .  .  .  .  .  .  .  .  .  Name: "int"
    51  .  .  .  .  .  .  .  .  }
    52  .  .  .  .  .  .  .  }
    53  .  .  .  .  .  .  .  Data: 0
    54  .  .  .  .  .  .  }
    55  .  .  .  .  .  }
    56  .  .  .  .  }
    57  .  .  .  .  OpPos: ./foo.z3:9:13
    58  .  .  .  .  Op: ==
    59  .  .  .  .  Y: *ast.BasicLit {
    60  .  .  .  .  .  ValuePos: ./foo.z3:9:16
    61  .  .  .  .  .  Kind: INT
    62  .  .  .  .  .  Value: "24"
    63  .  .  .  .  }
    64  .  .  .  }
    65  .  .  }
    66  .  .  Ellipsis: -
    67  .  .  Rparen: ./foo.z3:9:18
    68  .  }
    69  }
ExprStmt!
     0  *ast.ExprStmt {
     1  .  X: *ast.CallExpr {
     2  .  .  Fun: *ast.Ident {
     3  .  .  .  NamePos: ./foo.z3:10:2
     4  .  .  .  Name: "assert"
     5  .  .  }
     6  .  .  Lparen: ./foo.z3:10:8
     7  .  .  Args: []ast.Expr (len = 1) {
     8  .  .  .  0: *ast.BinaryExpr {
     9  .  .  .  .  X: *ast.BinaryExpr {
    10  .  .  .  .  .  X: *ast.Ident {
    11  .  .  .  .  .  .  NamePos: ./foo.z3:10:9
    12  .  .  .  .  .  .  Name: "x"
    13  .  .  .  .  .  .  Obj: *ast.Object {
    14  .  .  .  .  .  .  .  Kind: var
    15  .  .  .  .  .  .  .  Name: "x"
    16  .  .  .  .  .  .  .  Decl: *ast.ValueSpec {
    17  .  .  .  .  .  .  .  .  Names: []*ast.Ident (len = 1) {
    18  .  .  .  .  .  .  .  .  .  0: *ast.Ident {
    19  .  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:4:6
    20  .  .  .  .  .  .  .  .  .  .  Name: "x"
    21  .  .  .  .  .  .  .  .  .  .  Obj: *(obj @ 13)
    22  .  .  .  .  .  .  .  .  .  }
    23  .  .  .  .  .  .  .  .  }
    24  .  .  .  .  .  .  .  .  Type: *ast.Ident {
    25  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:4:8
    26  .  .  .  .  .  .  .  .  .  Name: "int"
    27  .  .  .  .  .  .  .  .  }
    28  .  .  .  .  .  .  .  }
    29  .  .  .  .  .  .  .  Data: 0
    30  .  .  .  .  .  .  }
    31  .  .  .  .  .  }
    32  .  .  .  .  .  OpPos: ./foo.z3:10:10
    33  .  .  .  .  .  Op: -
    34  .  .  .  .  .  Y: *ast.Ident {
    35  .  .  .  .  .  .  NamePos: ./foo.z3:10:11
    36  .  .  .  .  .  .  Name: "y"
    37  .  .  .  .  .  .  Obj: *ast.Object {
    38  .  .  .  .  .  .  .  Kind: var
    39  .  .  .  .  .  .  .  Name: "y"
    40  .  .  .  .  .  .  .  Decl: *ast.ValueSpec {
    41  .  .  .  .  .  .  .  .  Names: []*ast.Ident (len = 1) {
    42  .  .  .  .  .  .  .  .  .  0: *ast.Ident {
    43  .  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:5:6
    44  .  .  .  .  .  .  .  .  .  .  Name: "y"
    45  .  .  .  .  .  .  .  .  .  .  Obj: *(obj @ 37)
    46  .  .  .  .  .  .  .  .  .  }
    47  .  .  .  .  .  .  .  .  }
    48  .  .  .  .  .  .  .  .  Type: *ast.Ident {
    49  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:5:8
    50  .  .  .  .  .  .  .  .  .  Name: "int"
    51  .  .  .  .  .  .  .  .  }
    52  .  .  .  .  .  .  .  }
    53  .  .  .  .  .  .  .  Data: 0
    54  .  .  .  .  .  .  }
    55  .  .  .  .  .  }
    56  .  .  .  .  }
    57  .  .  .  .  OpPos: ./foo.z3:10:13
    58  .  .  .  .  Op: ==
    59  .  .  .  .  Y: *ast.BasicLit {
    60  .  .  .  .  .  ValuePos: ./foo.z3:10:16
    61  .  .  .  .  .  Kind: INT
    62  .  .  .  .  .  Value: "2"
    63  .  .  .  .  }
    64  .  .  .  }
    65  .  .  }
    66  .  .  Ellipsis: -
    67  .  .  Rparen: ./foo.z3:10:17
    68  .  }
    69  }
ExprStmt!
     0  *ast.ExprStmt {
     1  .  X: *ast.CallExpr {
     2  .  .  Fun: *ast.Ident {
     3  .  .  .  NamePos: ./foo.z3:11:2
     4  .  .  .  Name: "assert"
     5  .  .  }
     6  .  .  Lparen: ./foo.z3:11:8
     7  .  .  Args: []ast.Expr (len = 3) {
     8  .  .  .  0: *ast.BinaryExpr {
     9  .  .  .  .  X: *ast.Ident {
    10  .  .  .  .  .  NamePos: ./foo.z3:11:9
    11  .  .  .  .  .  Name: "x"
    12  .  .  .  .  .  Obj: *ast.Object {
    13  .  .  .  .  .  .  Kind: var
    14  .  .  .  .  .  .  Name: "x"
    15  .  .  .  .  .  .  Decl: *ast.ValueSpec {
    16  .  .  .  .  .  .  .  Names: []*ast.Ident (len = 1) {
    17  .  .  .  .  .  .  .  .  0: *ast.Ident {
    18  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:4:6
    19  .  .  .  .  .  .  .  .  .  Name: "x"
    20  .  .  .  .  .  .  .  .  .  Obj: *(obj @ 12)
    21  .  .  .  .  .  .  .  .  }
    22  .  .  .  .  .  .  .  }
    23  .  .  .  .  .  .  .  Type: *ast.Ident {
    24  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:4:8
    25  .  .  .  .  .  .  .  .  Name: "int"
    26  .  .  .  .  .  .  .  }
    27  .  .  .  .  .  .  }
    28  .  .  .  .  .  .  Data: 0
    29  .  .  .  .  .  }
    30  .  .  .  .  }
    31  .  .  .  .  OpPos: ./foo.z3:11:11
    32  .  .  .  .  Op: ==
    33  .  .  .  .  Y: *ast.Ident {
    34  .  .  .  .  .  NamePos: ./foo.z3:11:14
    35  .  .  .  .  .  Name: "y"
    36  .  .  .  .  .  Obj: *ast.Object {
    37  .  .  .  .  .  .  Kind: var
    38  .  .  .  .  .  .  Name: "y"
    39  .  .  .  .  .  .  Decl: *ast.ValueSpec {
    40  .  .  .  .  .  .  .  Names: []*ast.Ident (len = 1) {
    41  .  .  .  .  .  .  .  .  0: *ast.Ident {
    42  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:5:6
    43  .  .  .  .  .  .  .  .  .  Name: "y"
    44  .  .  .  .  .  .  .  .  .  Obj: *(obj @ 36)
    45  .  .  .  .  .  .  .  .  }
    46  .  .  .  .  .  .  .  }
    47  .  .  .  .  .  .  .  Type: *ast.Ident {
    48  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:5:8
    49  .  .  .  .  .  .  .  .  Name: "int"
    50  .  .  .  .  .  .  .  }
    51  .  .  .  .  .  .  }
    52  .  .  .  .  .  .  Data: 0
    53  .  .  .  .  .  }
    54  .  .  .  .  }
    55  .  .  .  }
    56  .  .  .  1: *ast.Ident {
    57  .  .  .  .  NamePos: ./foo.z3:11:17
    58  .  .  .  .  Name: "implies"
    59  .  .  .  }
    60  .  .  .  2: *ast.BinaryExpr {
    61  .  .  .  .  X: *ast.Ident {
    62  .  .  .  .  .  NamePos: ./foo.z3:11:26
    63  .  .  .  .  .  Name: "z"
    64  .  .  .  .  .  Obj: *ast.Object {
    65  .  .  .  .  .  .  Kind: var
    66  .  .  .  .  .  .  Name: "z"
    67  .  .  .  .  .  .  Decl: *ast.ValueSpec {
    68  .  .  .  .  .  .  .  Names: []*ast.Ident (len = 1) {
    69  .  .  .  .  .  .  .  .  0: *ast.Ident {
    70  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:7:6
    71  .  .  .  .  .  .  .  .  .  Name: "z"
    72  .  .  .  .  .  .  .  .  .  Obj: *(obj @ 64)
    73  .  .  .  .  .  .  .  .  }
    74  .  .  .  .  .  .  .  }
    75  .  .  .  .  .  .  .  Type: *ast.Ident {
    76  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:7:8
    77  .  .  .  .  .  .  .  .  Name: "bool"
    78  .  .  .  .  .  .  .  }
    79  .  .  .  .  .  .  }
    80  .  .  .  .  .  .  Data: 0
    81  .  .  .  .  .  }
    82  .  .  .  .  }
    83  .  .  .  .  OpPos: ./foo.z3:11:28
    84  .  .  .  .  Op: ==
    85  .  .  .  .  Y: *ast.Ident {
    86  .  .  .  .  .  NamePos: ./foo.z3:11:31
    87  .  .  .  .  .  Name: "true"
    88  .  .  .  .  }
    89  .  .  .  }
    90  .  .  }
    91  .  .  Ellipsis: -
    92  .  .  Rparen: ./foo.z3:11:35
    93  .  }
    94  }
ExprStmt!
     0  *ast.ExprStmt {
     1  .  X: *ast.CallExpr {
     2  .  .  Fun: *ast.Ident {
     3  .  .  .  NamePos: ./foo.z3:12:2
     4  .  .  .  Name: "assert"
     5  .  .  }
     6  .  .  Lparen: ./foo.z3:12:8
     7  .  .  Args: []ast.Expr (len = 3) {
     8  .  .  .  0: *ast.Ident {
     9  .  .  .  .  NamePos: ./foo.z3:12:9
    10  .  .  .  .  Name: "x"
    11  .  .  .  .  Obj: *ast.Object {
    12  .  .  .  .  .  Kind: var
    13  .  .  .  .  .  Name: "x"
    14  .  .  .  .  .  Decl: *ast.ValueSpec {
    15  .  .  .  .  .  .  Names: []*ast.Ident (len = 1) {
    16  .  .  .  .  .  .  .  0: *ast.Ident {
    17  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:4:6
    18  .  .  .  .  .  .  .  .  Name: "x"
    19  .  .  .  .  .  .  .  .  Obj: *(obj @ 11)
    20  .  .  .  .  .  .  .  }
    21  .  .  .  .  .  .  }
    22  .  .  .  .  .  .  Type: *ast.Ident {
    23  .  .  .  .  .  .  .  NamePos: ./foo.z3:4:8
    24  .  .  .  .  .  .  .  Name: "int"
    25  .  .  .  .  .  .  }
    26  .  .  .  .  .  }
    27  .  .  .  .  .  Data: 0
    28  .  .  .  .  }
    29  .  .  .  }
    30  .  .  .  1: *ast.Ident {
    31  .  .  .  .  NamePos: ./foo.z3:12:12
    32  .  .  .  .  Name: "iff"
    33  .  .  .  }
    34  .  .  .  2: *ast.UnaryExpr {
    35  .  .  .  .  OpPos: ./foo.z3:12:17
    36  .  .  .  .  Op: !
    37  .  .  .  .  X: *ast.Ident {
    38  .  .  .  .  .  NamePos: ./foo.z3:12:18
    39  .  .  .  .  .  Name: "z"
    40  .  .  .  .  .  Obj: *ast.Object {
    41  .  .  .  .  .  .  Kind: var
    42  .  .  .  .  .  .  Name: "z"
    43  .  .  .  .  .  .  Decl: *ast.ValueSpec {
    44  .  .  .  .  .  .  .  Names: []*ast.Ident (len = 1) {
    45  .  .  .  .  .  .  .  .  0: *ast.Ident {
    46  .  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:7:6
    47  .  .  .  .  .  .  .  .  .  Name: "z"
    48  .  .  .  .  .  .  .  .  .  Obj: *(obj @ 40)
    49  .  .  .  .  .  .  .  .  }
    50  .  .  .  .  .  .  .  }
    51  .  .  .  .  .  .  .  Type: *ast.Ident {
    52  .  .  .  .  .  .  .  .  NamePos: ./foo.z3:7:8
    53  .  .  .  .  .  .  .  .  Name: "bool"
    54  .  .  .  .  .  .  .  }
    55  .  .  .  .  .  .  }
    56  .  .  .  .  .  .  Data: 0
    57  .  .  .  .  .  }
    58  .  .  .  .  }
    59  .  .  .  }
    60  .  .  }
    61  .  .  Ellipsis: -
    62  .  .  Rparen: ./foo.z3:12:19
    63  .  }
    64  }
ExprStmt!
*/
