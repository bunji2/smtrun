#!/bin/sh
(
    echo "package main";
    echo "// VERSION : version of program";
    echo -n "const ";
    cat ./VERSION.txt
) > version.go
go build -o smtrun
