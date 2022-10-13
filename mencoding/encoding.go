package mencoding

import (
	"bufio"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
)

const (
	GBK     = "GBK"
	UTF8    = "UTF8"
	UNKNOWN = "UNKNOWN"
)

func Decoder(r *bufio.Reader) ([]byte, error) {
	reader := transform.NewReader(r, simplifiedchinese.GBK.NewDecoder())
	return ioutil.ReadAll(reader)
}

func Byte2Utf8(data []byte) ([]byte, error) {
	e := GetStrCoding(data)
	switch e {
	case GBK:
		return simplifiedchinese.GBK.NewDecoder().Bytes(data)
	default:
		return data, nil
	}
}

// DetermineEncoding 检测编码
//func DetermineEncoding(r *bufio.Reader) encoding.Encoding {
//	b, err := r.Peek(1024)
//	if err != nil && err.Error() != "EOF" {
//		return unicode.UTF8
//	}
//	e, name, _ := charset.DetermineEncoding(b, "")
//	fmt.Println(name)
//	for _, se := range simplifiedchinese.All {
//		if e == se {
//
//		}
//	}
//	return e
//}

func GetStrCoding(data []byte) string {
	if isUtf8(data) {
		return UTF8
	} else if isGBK(data) {
		return GBK
	} else {
		return UNKNOWN
	}
}

func isGBK(data []byte) bool {
	for i := 0; i < len(data); {
		if data[i] <= 0x7f { // 编码0-127
			i++
			continue
		} else {
			if data[i] >= 0x81 && data[i] <= 0xfe && data[i+1] >= 0x40 && data[i+1] <= 0xfe && data[i+1] != 0xf7 {
				i += 2
				continue
			} else {
				return false
			}
		}
	}
	return true
}

func preNum(data byte) int {
	var mask byte = 0x80
	var num = 0
	for i := 0; i < 8; i++ {
		if (data & mask) == mask {
			num++
			mask = mask >> 1
		} else {
			break
		}
	}
	return num
}

func isUtf8(data []byte) bool {
	for i := 0; i < len(data); {
		if (data[i] & 0x80) == 0x00 {
			i++
			continue
		} else if num := preNum(data[i]); num > 2 {
			i++
			for j := 0; j < num-1; j++ {
				if (data[i] & 0xc0) != 0x80 {
					return false
				}
				i++
			}
		} else {
			return false
		}
	}
	return true
}
