# smtrun --- front-end of go-z3

go-z3 のフロントエンド言語 SMTL を実装してみた。実装の特徴として Golang の ast パッケージを利用してみた。

なんで最初から Python を使わないのか？ Golang の AST を使ってみたかったから。

## SMT とは

SMT は "Satisfiable Modulo Theories" の略であり、充足可能性の判定を行う手法の一つである。
特徴として、一階述語論理式を扱うことができる。このような充足可能性を判定するものを SMT Solver と呼ぶ。

ここでは SMT Solver として [go-z3](https://github.com/mitchellh/go-z3) を使用した。

## 使い方

まずは最初に使い方を見てもらったほうがわかりやすい。

次のような条件式が与えられているときに、x と y を自動的に求めてみる。

![](https://latex.codecogs.com/gif.latex?x&plus;y=24\wedge{x-y=2})


SMTL では上を次のように記述する。

```
package smtl

func main() {
	var x int
	var y int
	assert(x+y == 24)
	assert(x-y == 2)
}
```

上のテキストを "foo.smtl" というファイル名で保存しておく。

実際に変数 x と y の解決は "smtrun" コマンドで求める。

```
% smtrun foo.smtl
x = 13
y = 11
```

## 数独の例

3 x 3 の数独を解く例を示す。

```
4│□│□
─┼─┼─
□│□│7
─┼─┼─
□│□│□
```

条件は以下の通りである。

* 各マス目には 1 〜 9 の異なる数字が入る。
* 各マス目の縦・横・斜めの合計はいずれも 15 となる。

```
// sudoku.smtl
// 3x3 の数独を解く例

package smtl

func main() {
  // 各マス目の並び
	// c00 c01 c02
	// c10 c11 c12
	// c20 c21 c22

	var c00, c01, c02 int
	var c10, c11, c12 int
	var c20, c21, c22 int

	// 値の範囲
	assert(c00>=1 && c00<=9)
	assert(c01>=1 && c01<=9)
	assert(c02>=1 && c02<=9)
	assert(c10>=1 && c10<=9)
	assert(c11>=1 && c11<=9)
	assert(c12>=1 && c12<=9)
	assert(c20>=1 && c20<=9)
	assert(c21>=1 && c21<=9)
	assert(c22>=1 && c22<=9)

	// c00 〜 c22 は一意な値
	assert(distinct(c00, c01, c02, c10, c11, c12, c20, c21, c22))

	// 判明している値
	assert(c00 == 4)
	assert(c12 == 7)

	// 縦の合計=15
	assert(c00+c10+c20 == 15)
	assert(c01+c11+c21 == 15)
	assert(c02+c12+c22 == 15)

	// 横の合計=15
	assert(c00+c01+c02 == 15)
	assert(c10+c11+c12 == 15)
	assert(c20+c21+c22 == 15)

	// 斜めの合計=15
	assert(c00+c11+c22 == 15)
	assert(c02+c11+c20 == 15)

}
```

smtrun コマンドを実行すると次のようになる。

```
% smtrun sudoku.smtl 
c00 = 4
c01 = 9
c02 = 2
c10 = 3
c11 = 5
c12 = 7
c20 = 8
c21 = 1
c22 = 6
```

解答は次の通りである。

```
4│9│2
─┼─┼─
3│5│7
─┼─┼─
8│1│6
```


## SMTL について

SMTL (SMT Language) の構文は Golang に類似するが、使用できる文や演算子が限定されている。

例えば SMTL において、if 文、for 文、代入文などの多くのプログラミングで用意されている構文は存在しないし、二項演算子の "/" や文字列を扱うための構文もない。

SMTL の BNF を以下に示す。

```
program
  := package main_function

package
  := "package" "smtl"

main_function
  := "func" "main" "(" ")" "{" statement_list "}"

statement_list
  := statement
  |  statement statement_list

statement
  := "var" identifier type
  |  assertion

assertion
  := "assert" "(" expression ")"

expression
  := "distinct" "(" identifier_list ")"
  |  expr

expr
  := "true"
  |  "false"
  |  identifier
  |  int_lit
  |  expr binary_op expr
  |  unary_op expr
  |  expr "." "implies" "(" expr ")"
  |  expr "." "iff" "(" expr ")"
  |  "(" expr ")"

identifier_list
  := identifier
	|  identifier "," identifier_list

binary_op
  := "+"
  |  "-"
  |  "*"
  |  "=="
  |  "!="
  |  ">"
  |  "<"
  |  ">="
  |  "<="

unary_op
  := "!"

Definitions of identifier and int_lit are according to Golang syntax definition.
Refer:
  https://golang.org/ref/spec#Identifiers
  https://golang.org/ref/spec#Integer_literals
```

## ビルド方法

開発環境は arm の debian を使用したが、intel の linux でもほぼ同様と思われる。

### go-z3 のビルド

事前に gcc、g++、python、そして golang をインストールしておくこと。

```
% go get github.com/mitchellh/go-z3
% cd $GOPATH/src/github.com/mitchellh/go-z3
```

make の前に、少しソースの修正が必要となる。
z3.go の cgo 宣言にの行を vi などのエディタで次のように修正する。

```
修正前：
// #cgo LDFLAGS: ${SRCDIR}/libz3.a -lstdc++

修正後：
// #cgo LDFLAGS: ${SRCDIR}/libz3.a -lstdc++ -lm
```

あとは go-z3 の手順通り make すればよい。

```
% make
```


詳細は [go-z3](https://github.com/mitchellh/go-z3) を参照のこと。


### smtrun コマンドのビルド

```
% go get github.com/bunji2/smtrun
% go build github.com/bunji2/smtrun
```
