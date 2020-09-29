package aesgcm

import (
	"testing"

	"github.com/ahui2016/recoit/util"
)

func TestEncryptDecrypt(t *testing.T) {
	passphrase := util.TimeNow() + util.NewID()
	plaintext := util.TimeNow() + util.NewID()
	key := NewKey(passphrase)
	gcm := NewGCM(key)

	ciphertext := gcm.Encrypt([]byte(plaintext))
	decryptText, err := gcm.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(plaintext)
	t.Log(string(decryptText))

	if string(decryptText) != plaintext {
		t.Fatal("decryptText is not equal to plaintext")
	}
}
