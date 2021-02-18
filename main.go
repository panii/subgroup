package main

import (
    "encoding/base64"
    //"encoding/hex"
    "encoding/json"

    "crypto/hmac"
	"crypto/sha256"
	//"crypto/md5"

    "sync"
    "bytes"
    "log"
    "strings"
    "strconv"
    "fmt"
    "net"
    "time"
    "regexp"

    "io/ioutil"
    "net/http"
)

const DingdingWebhookPatten string = `^https\:\/\/oapi\.dingtalk\.com\/robot\/send\?access\_token\=([a-z0-9]{64})$`
const FeishuWebhookPatten string = `^https\:\/\/open\.feishu\.cn\/open\-apis\/bot\/v2\/hook\/([a-z0-9]{8}\-[a-z0-9]{4}\-[a-z0-9]{4}\-[a-z0-9]{4}\-[a-z0-9]{12})$`