package ean

import (
	"testing"
)

func Test_EncodeEAN(t *testing.T) {
	code, err := GenerateEAN13(5, 1500)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(code)
}

func Test_DecodeEAN(t *testing.T) {
	productID, price, err := ParseEAN13("2000005015007")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(productID, price)
	if productID != 5 {
		t.Fatal("productID not match")
	}
	if price != 1500 {
		t.Fatal("price not match")
	}
}
