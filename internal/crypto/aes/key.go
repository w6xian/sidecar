package aes

// AES-128, AES-192, or AES-256 key length
var extraKey = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

// 16, 24, 32 bytes
func GetAES128Key(key []byte) []byte {
	l := len(key)
	if l >= 16 {
		return key[:16]
	}
	// 拼接key，满足16字节长度
	key = append(key, extraKey...)
	return key[:16]
}
func GetAES192Key(key []byte) []byte {
	l := len(key)
	if l >= 24 {
		return key[:24]
	}
	// 拼接key，满足24字节长度
	key = append(key, extraKey...)
	return key[:24]
}
func GetAES256Key(key []byte) []byte {
	l := len(key)
	if l >= 32 {
		return key[:32]
	}
	// 拼接key，满足32字节长度
	key = append(key, extraKey...)
	return key[:32]
}
