package aes

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
)

func AESEBCEncrypt(p, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	p = PKCS5Padding(p, block.BlockSize())
	decrypted := make([]byte, len(p))
	size := block.BlockSize()

	for bs, be := 0, size; bs < len(p); bs, be = bs+size, be+size {
		block.Encrypt(decrypted[bs:be], p[bs:be])
	}

	return decrypted, nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// Base64AESCBCEncrypt encrypts data with AES algorithm in CBC mode and encoded by base64
func Base64AESEBCEncrypt(p, key []byte) (string, error) {
	c, err := AESEBCEncrypt(p, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(c), nil
}

func AESEBCDecrypt(c, key []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	decrypted := make([]byte, len(c))
	size := block.BlockSize()

	for bs, be := 0, size; bs < len(c); bs, be = bs+size, be+size {
		block.Decrypt(decrypted[bs:be], c[bs:be])
	}

	return PKCS5UnPadding(decrypted), nil
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

// Base64AESCBCDecrypt decrypts cipher text encoded by base64 with AES algorithm in CBC mode
func Base64AESEBCDecrypt(c string, key []byte) ([]byte, error) {
	oriCipher, err := base64.StdEncoding.DecodeString(c)
	if err != nil {
		return nil, err
	}
	p, err := AESEBCDecrypt(oriCipher, key)
	if err != nil {
		return nil, err
	}
	return p, nil
}
