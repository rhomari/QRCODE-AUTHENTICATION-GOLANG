package qrcodepkg

import (
	"github.com/skip2/go-qrcode"
)

func MakeQRCode(data []byte) ([]byte, error) {
	qrimage, err := qrcode.Encode(string(data), qrcode.Medium, 256)
	return qrimage, err
}
