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