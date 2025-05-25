package main

import (
	"golang.org/x/text/encoding/traditionalchinese"
)

func decodeBIG5(byteString []byte) (string, error) {
	utfBytes, err := traditionalchinese.Big5.NewDecoder().Bytes(byteString)
	if err != nil {
		println("Fail to convert big5 to utf-8. ", err.Error())
		return "", err
	}
	return string(utfBytes), nil
}
