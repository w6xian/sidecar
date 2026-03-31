package ED25519

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
)

var (
	ErrKeyMustBePEMEncoded = errors.New("invalid key: Key must be a PEM encoded PKCS1 or PKCS8 key")
	ErrNotRSAPrivateKey    = errors.New("key is not a valid RSA private key")
	ErrNotRSAPublicKey     = errors.New("key is not a valid RSA public key")
)

type EDKey struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	err        error
}

func NewEDKey() *EDKey {
	return &EDKey{}
}

func FromPemKey(priPem string) *EDKey {

	k := &EDKey{}

	// 读取私钥文件
	privateKeyPEM, err := os.ReadFile(priPem)
	if err != nil {
		k.err = err
		return k
	}

	// Parse PEM block
	var block *pem.Block
	if block, _ = pem.Decode(privateKeyPEM); block == nil {
		k.err = errors.New("invalid key: Key must be a PEM encoded PKCS1 or PKCS8 key")
	}

	// Parse the key
	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
		k.err = err
	}

	var pkey ed25519.PrivateKey
	var ok bool
	if pkey, ok = parsedKey.(ed25519.PrivateKey); !ok {
		k.err = errors.New("invalid key: Key must be a PEM encoded ED key")
	}
	k.PrivateKey = pkey
	return k
}

func FromPublicPemKey(pubPem string) *EDKey {

	k := &EDKey{}

	// 读取私钥文件
	publicKeyPEM, err := os.ReadFile(pubPem)
	if err != nil {
		k.err = err
		return k
	}

	fmt.Println(string(publicKeyPEM))

	// Parse PEM block
	var block *pem.Block
	if block, _ = pem.Decode(publicKeyPEM); block == nil {
		k.err = errors.New("invalid key: Key must be a PEM encoded PKCS1 or PKCS8 key")
	}

	fmt.Println(block.Bytes)

	// Parse the key
	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		fmt.Println(err.Error())
		k.err = err
	}

	var pkey ed25519.PublicKey
	var ok bool
	if pkey, ok = parsedKey.(ed25519.PublicKey); !ok {
		k.err = errors.New("invalid key: Key must be a PEM encoded ED key")
	}
	k.PublicKey = pkey
	return k
}

func DecodePublicPemKey(pubPem string) (ed25519.PublicKey, error) {
	// 读取私钥文件
	publicKeyPEM, err := os.ReadFile(pubPem)
	if err != nil {
		return nil, err
	}

	return DecodePublicPemString(publicKeyPEM)

}

func DecodePublicPemString(publicKeyPEM []byte) (ed25519.PublicKey, error) {

	// Parse PEM block
	var block *pem.Block
	if block, _ = pem.Decode(publicKeyPEM); block == nil {
		return nil, errors.New("invalid key: Key must be a PEM encoded PKCS1 or PKCS8 key")
	}

	// Parse the key
	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	var pkey ed25519.PublicKey
	var ok bool
	if pkey, ok = parsedKey.(ed25519.PublicKey); !ok {
		return nil, errors.New("invalid key: Key must be a PEM encoded ED key")
	}
	return pkey, nil
}

func (k *EDKey) Valid() {}

func (k *EDKey) Keys() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		k.err = err
		return nil, nil, err
	}
	k.PublicKey = publicKey
	k.PrivateKey = privateKey
	k.err = err
	return publicKey, privateKey, err
}
func (k *EDKey) HasError() bool {
	return k.err != nil
}

func (k *EDKey) Save() error {
	p := "./"
	if k.HasError() {
		fmt.Println(k.err.Error())
		return k.err
	}
	// x509格式封装
	x509PrivateKey, err := x509.MarshalPKCS8PrivateKey(k.PrivateKey) // ed25519的这里privateKey不是一个指针

	if err != nil {
		fmt.Println(err)
		return err
	}
	x509PublicKey, err := x509.MarshalPKIXPublicKey(k.PublicKey)
	if err != nil {
		log.Println(err)
		return err
	}

	// 设置pem编码的数据块
	privateBlock := &pem.Block{
		Type:  "PRIVATE_KEY",
		Bytes: x509PrivateKey,
	}
	privateKeyFile := p + "private.pem"
	privateFile, err := os.OpenFile(privateKeyFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer privateFile.Close()
	pem.Encode(privateFile, privateBlock) // pem编码

	publicBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509PublicKey,
	}
	publicKeyFile := p + "public.pem"
	publicFile, err := os.OpenFile(publicKeyFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer publicFile.Close()
	pem.Encode(publicFile, publicBlock)
	return nil
}
