package commoncontrollers

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/dgrijalva/jwt-go"
)

type BaseController struct {
	beego.Controller
	controllerName string
	actionName     string
}

// 初始化函数
func (this *BaseController) Prepare() {
	controllerName, actionName := this.GetControllerAndAction()
	this.controllerName = strings.ToLower(controllerName[0 : len(controllerName)-10])
	this.actionName = strings.ToLower(actionName)

	if this.Ctx.Request.Method == http.MethodOptions {
		this.StopRun()
	}

	// beego.Debug("---------------request method:", this.Ctx.Request.Method)
	// beego.Debug("---------------request body:", string(this.Ctx.Input.RequestBody))

	// pass login
	if controllerName == "AdminController" && actionName == "Login" {
		return
	}
	if this.Ctx.Input.Header("Authorization") == "" {
		beego.Error("Authorization is nil")
		this.ResponseAuth(1001, "Authorization is nil", nil)
		this.StopRun()
	}

	token := strings.TrimSpace(strings.TrimLeft(this.Ctx.Input.Header("Authorization"), "Bearer"))
	if token == "" {
		beego.Error("unknown token:", token)
		this.ResponseAuth(1001, "unknown token", nil)
		this.StopRun()
	}

	err := this.TokenAuth(token)
	if err != nil {
		beego.Error(err)
		this.ResponseAuth(1001, err.Error(), nil)
		this.StopRun()
	}

}

// 响应json
func (this *BaseController) ResponseJson(isSuccess bool, msg string, data map[string]interface{}) {
	status := 1
	if isSuccess {
		status = 0
	}
	ret := map[string]interface{}{"code": status, "msg": msg}
	if data != nil {
		ret["data"] = data
	} else {
		ret["data"] = map[string]interface{}{}
	}

	this.Data["json"] = ret
	this.ServeJSON()
}

// 响应json
func (this *BaseController) ResponseAuth(code int, msg string, data map[string]interface{}) {
	ret := map[string]interface{}{"code": code, "msg": msg}
	if data != nil {
		ret["data"] = data
	}

	this.Data["json"] = ret
	this.ServeJSON()
}

// --------------------Authorization Token--------------------
type Claims struct {
	App string `json:"app"`
	// recommended having
	jwt.StandardClaims
}

func (this *BaseController) GetToken(username, domain string) (string, error) {
	var ttl int64 = 3600
	// current timestamp
	currentTimestamp := time.Now().UTC().Unix()
	// md5 of sub & iat
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(username))
	md5Ctx.Write([]byte(strconv.FormatInt(int64(currentTimestamp), 10)))
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		App: beego.AppConfig.String("appname"),
		StandardClaims: jwt.StandardClaims{
			Subject:   username,
			IssuedAt:  currentTimestamp,
			ExpiresAt: currentTimestamp + ttl,
			NotBefore: currentTimestamp,
			Issuer:    domain,
			Id:        string(md5Ctx.Sum(nil)),
		},
	})
	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(beego.AppConfig.String("HMACKEY")))
	if err != nil {
		beego.Error(err)
		return "", err
	}

	return tokenString, nil
}

func (this *BaseController) TokenAuth(tokenstring string) error {
	token, err := jwt.ParseWithClaims(tokenstring, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(beego.AppConfig.String("HMACKEY")), nil
	})
	if err != nil {
		return err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid && claims != nil {
		// succees
		this.Ctx.SetCookie("name", claims.Subject)
		return nil
	} else {
		return errors.New("Token Unauthorized")
	}
}
