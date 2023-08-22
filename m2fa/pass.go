package m2fa

import (
	"fmt"
	"github.com/pquerna/otp/totp"
	"time"
)

func m() {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "myapp",
		AccountName: "quiet_king0312@163.com",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("key:", key)

	otp, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("otp:", otp)

}
