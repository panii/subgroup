package main

import (
    "encoding/base64"
    //"encoding/hex"
    "encoding/json"

    "crypto/hmac"
	"crypto/sha256"
    "math/rand"
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

var Utils = &Util{}
var Memcache sync.Map

type CacheData struct {
	Ip string
    TimeAdded int64
}

// 检查首页输入的webhook缓存对象, 有过期就删除之
func checkCacheExpire() {
    //fmt.Println("start checkCacheExpire")
    //fmt.Printf("%d", time.Now().Unix())
    
    Memcache.Range(func (key, value interface{}) bool {
        cacheData := value.(*CacheData)
        //fmt.Printf("range %s %d", key, cacheData.TimeAdded)
        if (time.Now().Unix() - cacheData.TimeAdded) > 86400 { // webhook在内存里超过1天, 删除之。
            //fmt.Printf("delete %s", key)
            Memcache.Delete(key)
        }
        
        return true
    })
    
    //fmt.Println("end checkCacheExpire")
    time.Sleep(time.Second * 300) // 每5分钟检查一次有没有过期的webhook
    go checkCacheExpire()
}

func main() {
    //mux := http.NewServeMux()
    http.Handle("/", http.RedirectHandler("/webhook", 307))
    http.Handle("/webhook", webhookHandler())
    http.Handle("/detail", detailHandler())
    //http.Handle("/loadwebhook", loadwebhookHandler())
    //http.Handle("/sendmsg", sendmsgHandler())
    
    http.Handle("/time", timeHandler(time.RFC1123))
    http.Handle("/favicon.ico", iconHandler())

    log.Println("Listening 8091...")
    go checkCacheExpire()
    http.ListenAndServe(":8091", nil)
}

// router "/time"
func timeHandler(format string) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        time.Sleep(time.Duration(2) * time.Second)
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        
        defer req.Body.Close()
        str2, err2 := ioutil.ReadAll(req.Body) //获取post的数据
        if err2 != nil {
            fmt.Fprintf(w, "无法推送, 提交内容错误")
            return
        }
        
        var str = string(str2)
        //var str = `{"secretvalue":"secret","webhookvalue":"webhook","sendcontent":"今日饭搭子{number}组\n组织者：{2 from SubGroup B}{10 from SubGroup C}\n参与者：{1 from SubGroup A}\n\n\n... ","subgroups":["组员1\n组员2\n组员3","组员4\n组员5\n组员6","组员7\n组员8\n组员9","组员a\n组员b\n组员c"]}`
        //var str = `{"secretvalue":"secret","webhookvalue":"webhook","sendcontent":"今日饭搭子{number}组\n组织者：{1 from SubGroup A}|||{1 from SubGroup A}\n参与者：{20 from SubGroup A}\n\n... ","subgroups":["组员1\n组员2\n组员3","组员4\n组员5\n组员6","组员7\n组员8\n组员9","组员a\n组员b\n组员c"]}`
        
        m := make(map[string]interface{})
        err := json.Unmarshal([]byte(str), &m)
        if err != nil {
            fmt.Fprintf(w, "无法推送, 提交内容错误")
            return
        }
        //fmt.Println(m)
        rand.Seed(time.Now().Unix())
        
        if _, isset := m["subgroups"]; !isset {
            fmt.Fprintf(w, "无法推送, 请填写SubGroups")
            return
        }
        
        // todo 总数检查 不能太多subgroup
        var subGroups []interface{}
        subGroups = m["subgroups"].([]interface{})
        var subGroups_A []string
        var subGroups_B []string
        var subGroups_C []string
        var subGroups_D []string
        if len(subGroups) > 0 {
            subGroups_A = strings.Split(strings.Trim(subGroups[0].(string), " "), "\n")
            rand.Shuffle(len(subGroups_A), func(i, j int) {
                subGroups_A[i], subGroups_A[j] = subGroups_A[j], subGroups_A[i]
            })
        }
        if len(subGroups) > 1 {
            subGroups_B = strings.Split(strings.Trim(subGroups[1].(string), " "), "\n")
            rand.Shuffle(len(subGroups_B), func(i, j int) {
                subGroups_B[i], subGroups_B[j] = subGroups_B[j], subGroups_B[i]
            })
        }
        if len(subGroups) > 2 {
            subGroups_C = strings.Split(strings.Trim(subGroups[2].(string), " "), "\n")
            rand.Shuffle(len(subGroups_C), func(i, j int) {
                subGroups_C[i], subGroups_C[j] = subGroups_C[j], subGroups_C[i]
            })
        }
        if len(subGroups) > 3 {
            subGroups_D = strings.Split(strings.Trim(subGroups[3].(string), " "), "\n")
            rand.Shuffle(len(subGroups_D), func(i, j int) {
                subGroups_D[i], subGroups_D[j] = subGroups_D[j], subGroups_D[i]
            })
        }
        
        if _, isset := m["sendcontent"]; !isset {
            fmt.Fprintf(w, "无法推送, 推送内容错误")
            return
        }
        var sendcontentArr []string
        sendcontentArr = strings.Split(strings.Trim(strings.Trim(strings.Trim(strings.Replace(m["sendcontent"].(string), "\r\n", "\n", -1), "\n"), "\t"), " "), "\n")
        
        var length = len(sendcontentArr)
        if length == 0 {
            fmt.Fprintf(w, "无法推送, 推送内容错误")
            return
        }
        
        var repeat = false
        if sendcontentArr[length - 1] == "..." {
            if length == 1 {
                fmt.Fprintf(w, "无法推送, 推送内容错误")
                return
            }
            repeat = true
            
            sendcontentArr = sendcontentArr[0:(length - 1)]
        }

        var number = 1
        sendcontent := Utils.GenerateContent(repeat, number, sendcontentArr, subGroups_A, subGroups_B, subGroups_C, subGroups_D)

        if sendcontent == "" {
            fmt.Fprintf(w, "无法推送, 推送内容错误")
            return
        }
        
        
        // fmt.Println(subGroups_A)
        // fmt.Println(subGroups_B)
        // fmt.Println(subGroups_C)
        // fmt.Println(subGroups_D)
        fmt.Println()
        fmt.Println("'" + sendcontent + "'")
        
        if _, isset := m["webhookvalue"]; !isset {
            fmt.Fprintf(w, "无法推送, 请填写webhook和secret")
            return
        }
        if _, isset := m["secretvalue"]; !isset {
            fmt.Fprintf(w, "无法推送, 请填写webhook和secret")
            return
        }
        webhookurl := m["webhookvalue"].(string)
        secretvalue := m["secretvalue"].(string)
        
        if webhookurl == "" && secretvalue == "" {
            fmt.Fprintf(w, "无法推送, 请填写webhook和secret")
            return
        }
        
        matched, chatApp := GetChatApp(webhookurl, secretvalue)
        var sendResult bool
        var errMsg string
        if matched {
            sendResult, errMsg = chatApp.SendMsg(sendcontent)
        } else {
            fmt.Fprintf(w, "无法推送, 请填写正确的webhook和secret")
            return
        }
        
        
        //fmt.Fprintf(w, strconv.FormatBool(sendResult))
        //fmt.Fprintf(w, " ")
        if sendResult == true {
            fmt.Fprintf(w, "推送成功")
        } else {
            fmt.Fprintf(w, "推送失败")
            fmt.Fprintf(w, ", 远端返回: " + errMsg)
        }
        
        return
        
        //tm := time.Now().Format(format)
        //w.Write([]byte("The time is: " + tm))
    }
}

// 输出图标流
// router: "/favicon.ico"
func iconHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        icon := "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAABaUlEQVQ4T63TTUiUURTG8d9LC2ljkJNiJWgEQWokSEEk6TBGmh9ELgwh9+5auYhaFUZlGYJC5giVaAhCaS5KIwlCXERSgllBuA7d6mri9U2wSAZtzuZyuc/5n697glSHawJXkWV7tiblZpC6ZXUHzhuh1kJAanuB/1RnCHDuETlHGKxIn0xhdaT58Xr9iDJoGCZ2lOSx9IBLbyLNUPwvQMEZfs6Tlc1YCytfqe5h/8nIYa6PYBeVHdH9Qy/T7ZsyOHiad9dJdLEwwseHXJ5h9i6xEg6cYqSWmr4I8KqNpbebAPtK6S+maYLsAj4lid/jQQ6FCRqfkTxO4v4WJWwALr5gTxGfH1N1m669FJ2lcZiBMuKdBME/epB/gpetXBjl+zjzT2meZOoK4dvherpzqXtCXllUzvKX3yXUJMkvZ3csHAxDlSwv0jRO2JvQ3t9g9g6HzlM/yLfn6wEz9JHST39LRQaW6T/X+RcCOH9dQPJkUwAAAABJRU5ErkJggg=="
        resBytes, _ := base64.StdEncoding.DecodeString(icon)
        w.Header().Set("Content-Type", "image/png")
        w.Write(resBytes)
    }
}

// 首页: 只做一件事, 用户输入webhook
// router: "/webhook"
func webhookHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        switch req.Method {
            case "GET":
                // 返回首页html, 需要用户填写webhook
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                w.Write([]byte(`<title>SubGroup.today</title>
<link rel="preconnect" href="https://fonts.gstatic.com">
<link href="https://fonts.googleapis.com/css2?family=Dosis:wght@600&display=swap" rel="stylesheet">
<meta content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=yes" name="viewport" />
<body style="text-align:center">
<div id="main">
<style>
*{margin:0;padding:0}
#main{max-width:320px;margin:0 auto;overflow:hidden;transition:transform 0.5s;transform:translateY(168px);}
#intro h2 {
    color: #f66;
    font-size: 50px;

    line-height: 50px;
    margin-bottom: 5px;
    padding-left: 2px;
    padding-right: 2px;
    text-shadow: 0 1px 0 #cccccc, 1px 2px 0 #cccccc, 2px 3px 0 #cccccc;
    font-family: 'Dosis', sans-serif;
}
#intro h3 {
    font-size: 23px;
    line-height: 100%;
    margin-bottom: 30px;
    color: #666;
    text-shadow: 0 1px 0 #eeeeee, 1px 2px 0 #eeeeee, 2px 3px 0 #eeeeee;
    font-family: 'Dosis', sans-serif;
}
h4 {margin:5px auto 3px;color:#666;font-family: 'Dosis', Arial, sans-serif;text-align:left;}
input.btn1 {
    -webkit-tap-highlight-color: transparent; -webkit-appearance: none;
    -moz-border-radius: 58px;
    -webkit-border-radius: 58px;
    -o-border-radius: 58px;
    border-radius: 58px;
    -moz-box-shadow: 0 1px 0 0 #FFCCCC inset, 0 1px 0 0 #CCCCCC;
    -webkit-box-shadow: 0 1px 0 0 #ffcccc inset, 0 1px 0 0 #cccccc;
    -o-box-shadow: 0 1px 0 0 #FFCCCC inset, 0 1px 0 0 #CCCCCC;
    box-shadow: 0 1px 0 0 #ffcccc inset, 0 1px 0 0 #cccccc;
    background: -moz-linear-gradient(center top, #FF9999, #FF6666) repeat scroll 0 0 transparent;
    background: -webkit-linear-gradient(top, #f99, #f66) repeat scroll 0 0 #0000;
    background: -o-linear-gradient(top, #FF9999, #FF6666) repeat scroll 0 0 transparent;
    background: linear-gradient(top, #FF9999, #FF6666) repeat scroll 0 0 transparent;
    border: medium none;
    color: #fff;
    cursor: pointer;
    font-size: 15px;
    font-weight: bold;
    height: 60px;
    padding: 11px 15px;
    text-align: center;
    text-transform: uppercase;
    width: 60px;
    bottom: 18px;
    margin: 10px auto;
    opacity: 1;
    outline: none;
    user-select: none;
}
input.btn11 {
    border: medium none;
    -moz-box-shadow: 0 1px 0 0 #FFCCCC inset, 0 1px 0 0 #CCCCCC;
    -webkit-box-shadow: 0 1px 0 0 #ffcccc inset, 0 1px 0 0 #cccccc;
    -o-box-shadow: 0 1px 0 0 #FFCCCC inset, 0 1px 0 0 #CCCCCC;
    box-shadow: 0 1px 0 0 #ffcccc inset, 0 1px 0 0 #cccccc;
    background: -moz-linear-gradient(center top, #FFAA00, #FF6600) repeat scroll 0 0 transparent;
    background: -webkit-linear-gradient(top, #fa0, #f60) repeat scroll 0 0 #0000;
    background: -o-linear-gradient(top, #FFAA00, #FF6600) repeat scroll 0 0 transparent;
    background: linear-gradient(top, #FFAA00, #FF6600) repeat scroll 0 0 transparent;
}
#resultbox {
    margin-top: 10px;
}
#result {
    
}
.shine {
    animation: shine 0.4s ease infinite alternate;
}
@keyframes shine {
    0% {opacity: 0;}
    100% {opacity: 1;}
}

element {
   --main-placeholder-color: #666;
}
.heart {
    animation: heart 0.3s ease infinite alternate;
}
@keyframes heart {
    0% {--main-placeholder-color: #FFF;}
    100% {--main-placeholder-color: #F00;}
}

input::-webkit-input-placeholder, textarea::-webkit-input-placeholder {
  color: var(--main-placeholder-color);
  font-size: 14px;
}

input:-moz-placeholder, textarea:-moz-placeholder {
  color: var(--main-placeholder-color);
  font-size: 14px;
}

input::-moz-placeholder, textarea::-moz-placeholder {
  color: var(--main-placeholder-color);
  font-size: 14px;
}

input:-ms-input-placeholder, textarea:-ms-input-placeholder {
  color: var(--main-placeholder-color);
  font-size: 14px;
}

input.txt {
    border-radius:4px;border:1px solid #dee0e3;padding:8px;font-size:14px;width:310px;outline:none;margin-top:1px;
    transition:transform 0.2s;color:#666;
    -webkit-appearance: none;
}
.txt-hide {
    margin-top: -35px;
    text-align: left;
    height: 19px;
    line-height: 19px;
    font-family: 'Dosis', sans-serif;
    padding: 8px 0;
    transform: translateX(320px);
    transition: transform 0.2s;
}
#detail {
    height: 0px;
    overflow: hidden;
    transition: height 0.5s;
}
input.btn2, input.btn3 {
    width:93px;padding:6px 0px;color:#FFF;border-radius:4px;font-size:14px;cursor:pointer;
    -webkit-tap-highlight-color: transparent; -webkit-appearance: none;
}
#subgroups > div, #resultbox > div {
    transition: height 0.3s;
    height: 0px;
    overflow: hidden;
}
.subgroupbox {
    width: 310px;
    background-color: #fff;
    border: 1px solid #dee0e3;
    border-radius: 4px;
    margin: 0 auto;
    text-align: left;
    position: relative;
    min-height: 55px;
    padding: 9px 10px;
    box-sizing: border-box;
}
.subgroupbox-container {
    display: inline-block;
    max-width: 100%;
    width: 100%;
    line-height: 20px;
    padding: 2px 0;
    min-width: 20px;
    position: static;
}
.subgroupbox-wrap {
    max-height: 300px;
    overflow-y: scroll;
    font-variant-ligatures: none;
}
.subgroupbox-editor {
    outline: none;
    padding: 0;
    overflow-x: hidden;
    font-size: 14px;
    -webkit-font-smoothing: antialiased;
    font-variant-ligatures: none;
    color: #2b2f36;
}
.toolbar-item {
    display: inline-block;
    font-size: 26px;
    cursor: pointer;
    margin-top: 3px;
    user-select: none;
    -webkit-tap-highlight-color: transparent; -webkit-appearance: none;
}
.toolbar-item > div {
    display: inline-block;
    padding: 10px;
    transform: scale(0) translateX(0px);
    transition: transform 0.3s;
}
.larkc-svg-icon {
    width: 1em;
    height: 1em;
    vertical-align: -.15em;
    fill: #666;
    overflow: hidden;
    cursor: pointer;
    flex: none;
}
</style>
<div id="intro">
<h2>SubGroup.today</h2>
<h3>SubGroup in Group Chat</h3>
</div>

<script>
var detailDivClientHeight = "313px";
function backtohome() {
    var btn1 = document.getElementById("btn1");
    btn1.style.transition = "opacity 0.2s";
    btn1.style.opacity = "0";
    btn1.disabled = "disabled";
    btn1.className = "btn1";
    btn1.value = "继续";
    window.step = 1;
    document.getElementById("resultbox").style.display = "none";
    document.getElementById("resultbox").nextSibling.style.display = "none";
    var detaildiv = document.getElementById("detail");
    window.detailDivClientHeight = detaildiv.style.height = detaildiv.clientHeight + "px";
    setTimeout(function() {
        detaildiv.style.height = "0px";
    }, 20);
    setTimeout(function() {
        setTimeout(function(){
            document.getElementById("webhookvalue").style.transform = "translateX(0px)";
            document.getElementById("secretvalue").style.transform = "translateX(0px)";
            document.getElementById("webhookvaluehide").style.transform = "translateX(320px)";
            document.getElementById("secretvaluehide").style.transform = "translateX(320px)";
            document.getElementById("webhookvalue").removeAttribute("disabled");
            document.getElementById("secretvalue").removeAttribute("disabled");
            document.getElementById("webhookvalue").focus();
            document.getElementById("webhookvalue").select();
        }, 250);
        document.getElementById("main").style.transform = "translateY(168px)";
        setTimeout(function() {
            btn1.style.transition = "opacity 0.8s";
            btn1.style.opacity = "1";
            btn1.removeAttribute("disabled");
        }, 500);
    }, 150);
}
var step = 1;
function homecontinue() {
    if (window.step == 2) { doPushMsg(); return; }
    if (!formcheck()) return;
    if (window.detailHtml == "") {
        window.detailHtml = document.getElementById("detailHTMLTemplate").value;
        document.getElementById("detailHTMLTemplate").parentNode.innerHTML = window.detailHtml;
        setTimeout(function(){
            document.getElementById("addNewSubGroupBtn").click();
        }, 10);
    }

    var btn1 = document.getElementById("btn1");
    btn1.style.transition = "";
    btn1.style.opacity = "0";
    btn1.disabled = "disabled";
    btn1.className = "btn1 btn11";
    btn1.value = "推送";
    window.step = 2;
    document.getElementById("webhookvalue").disabled = "disabled";
    document.getElementById("secretvalue").disabled = "disabled";
    setTimeout(function() {
        btn1.style.transition = "opacity 0.8s";
        btn1.style.opacity = "1";
        btn1.removeAttribute("disabled");
        var detaildiv = document.getElementById("detail");
        setTimeout(function() {
            detaildiv.style.height = detailDivClientHeight;
            document.getElementById("main").style.transform = "translateY(3px)";
            setTimeout(function() {document.getElementById("secretvaluehide").style.transform = "translateX(0px) translateY(-15px)";}, 150);
            setTimeout(function() {
                detaildiv.style.height = "auto";
                document.getElementById("resultbox").style.display = "block";
                document.getElementById("resultbox").nextSibling.style.display = "inline-block";
            }, 500);
        }, 450);
    }, 200);
    document.getElementById("webhookvalue").style.transform = "translateX(-320px)";
    document.getElementById("secretvalue").style.transform = "translateX(-320px)";
    document.getElementById("webhookvaluehide").style.transform = "translateX(0px)";
    document.getElementById("secretvaluehide").style.transform = "translateX(0px)";
}
var oldplaceholder = {};
var timeid = {};
var botType = "";
function formcheck() {
    if (document.getElementById("webhookvaluehideTemplate")) {
        var webhookvaluehideTemplateHTML = document.getElementById("webhookvaluehideTemplate").value;
        document.getElementById("webhookvaluehideTemplate").parentNode.style.height = "auto";
        document.getElementById("webhookvaluehideTemplate").parentNode.innerHTML = webhookvaluehideTemplateHTML;
    }
    if (document.getElementById("secretvaluehideTemplate")) {
        var webhookvaluehideTemplateHTML = document.getElementById("secretvaluehideTemplate").value;
        document.getElementById("secretvaluehideTemplate").parentNode.innerHTML = webhookvaluehideTemplateHTML;
    }

    window.botType = "";
    document.getElementById("botType").innerHTML = window.botType;
    var str = document.form1.webhookvalue.value = document.form1.webhookvalue.value.trim();
    var str2 = document.form1.secretvalue.value = document.form1.secretvalue.value.trim();
    if (!str && !str2) return true;
    if (!str) {showPlaceholderError("1", document.form1.webhookvalue, "请填写Webhook"); return false;}
    if (!str2) {showPlaceholderError("2", document.form1.secretvalue, "请填写Secret"); return false;}
    var check1 = str.match(/` + DingdingWebhookPatten + `/);
    var check2 = str.match(/` + FeishuWebhookPatten + `/);
    if (check1 && check1[1]) {
        window.botType = "dingtalk.com";
        document.getElementById("botType").innerHTML = window.botType;
        return true;
    }
    if (check2 && check2[1]) {
        window.botType = "feishu.cn";
        document.getElementById("botType").innerHTML = window.botType;
        return true;
    }
    showPlaceholderError("1", document.form1.webhookvalue, "格式错误");
}
function showPlaceholderError(id, input, errmsg) {
    if (typeof window.oldplaceholder[id] == "undefined") window.oldplaceholder[id] = input.placeholder;
    input.value = "";
    input.placeholder = errmsg;
    input.className = "txt heart";
    clearTimeout(window.timeid[id]);
    window.timeid[id] = setTimeout(function(){input.placeholder = window.oldplaceholder[id]; input.className = "txt";}, 1500);    
    input.focus();
    return false;
}

var allGroups = ["A", "B", "C", "D"];
var addedGroups = [];
function addNewSubGroup(content) {
    if (allGroups.length == 0) return;
    var charr = allGroups.splice(0, 1)[0];
    addedGroups.unshift(charr);
    var html = ['<h4>SubGroup ' + charr + '</h4>',
                '<div class="subgroupbox">',
                  '<div class="subgroupbox-container">',
                    '<div class="subgroupbox-wrap">',
                        '<pre id="subgroupbox-editor-' + charr + '" spellcheck="false" class="subgroupbox-editor" contenteditable="true" onpaste="subgroupbox_editor_onpaste(event)" onkeydown="subgroupbox_editor_onkeydown(event)">' + content + '</pre>',
                    '</div>',
                  '</div>',
                '</div>'].join("");
    var subgroupdiv = document.createElement("div");
    subgroupdiv.innerHTML = html;
    subgroupdiv.style.height = "0px";
    document.getElementById("subgroups").append(subgroupdiv);
    subgroupdiv.setAttribute("id", "subgroup-" + charr);
    setTimeout(function(){ subgroupdiv.style.height = "112px"; }, 10);
    setTimeout(function(){ subgroupdiv.style.height = "auto" }, 300);
    toolbarItemAnimation();
}
function removeNewestSubGroup() {
    if (addedGroups.length == 0) return;
    var charr = addedGroups.splice(0, 1)[0];
    allGroups.unshift(charr);
    var subgroupdiv = document.getElementById("subgroup-" + charr);
    subgroupdiv.style.height = subgroupdiv.clientHeight + "px";
    setTimeout(function(){ subgroupdiv.style.height = "0px"; }, 10);
    setTimeout(function(){ document.getElementById("subgroups").removeChild(subgroupdiv); }, 300);
    toolbarItemAnimation();
}
function toolbarItemAnimation() {
    document.getElementById("addNewSubGroupBtn").style.transform = "scale(1) translateX(0px)";
    document.getElementById("removeNewestSubGroupBtn").style.transform = "scale(1) translateX(0px)";
    if (allGroups.length == 0) {
        document.getElementById("addNewSubGroupBtn").style.transform = "scale(0) translateX(0px)";
        document.getElementById("removeNewestSubGroupBtn").style.transform = "scale(1) translateX(-27px)";
    }
    if (addedGroups.length <= 1) {
        document.getElementById("addNewSubGroupBtn").style.transform = "scale(1) translateX(27px)";
        document.getElementById("removeNewestSubGroupBtn").style.transform = "scale(0) translateX(0px)";
    }
}
function doPushMsg() {
    var sendcontent = document.getElementById("subgroupbox-editor-sendcontent").innerText;
    var subgroups = [];
    var temp;
    for (var charr = 65; charr < 88; charr++) {
        temp = document.getElementById("subgroupbox-editor-" + String.fromCharCode(charr));
        if (!temp)
            break;
        subgroups.push(temp.innerText);
    }
    var btn1 = document.getElementById("btn1");
    btn1.style.transition = "opacity 0.06s";
    btn1.style.opacity = "0";
    btn1.disabled = "disabled";
    
    addNewestResult("正在推送...");
    fetchPushMsg(btn1, JSON.stringify({webhookvalue: document.form1.webhookvalue.value, secretvalue: document.form1.secretvalue.value, sendcontent: sendcontent, subgroups: subgroups}));
}
function addNewestResult(initialLogMsg) {
    var html = ['<h4>' + new Date().toLocaleString() + '</h4>',
                '<div class="subgroupbox">',
                  '<div class="subgroupbox-container">',
                    '<div class="subgroupbox-wrap">',
                        '<pre id="result" class="subgroupbox-editor shine" style="white-space:pre-wrap;word-wrap:break-word;">' + initialLogMsg + '</pre>',
                    '</div>',
                  '</div>',
                '</div>'].join("");
    var newestResultDiv = document.createElement("div");
    newestResultDiv.innerHTML = html;
    newestResultDiv.style.height = "0px";
    var resultdiv = document.getElementById("result");
    if (resultdiv) {
        resultdiv.className = "subgroupbox-editor";
        resultdiv.removeAttribute("id");
    }
    var resultbox = document.getElementById("resultbox");
    if (resultbox.children.length == 0) {
        resultbox.append(newestResultDiv);
    } else {
        resultbox.insertBefore(newestResultDiv, resultbox.children[0]);
    }
    setTimeout(function(){ newestResultDiv.style.height = "79px"; }, 10);
    setTimeout(function(){ newestResultDiv.style.height = "auto" }, 300);
    document.getElementById("removeOldestResultBtn").style.transform = "scale(0)";
}
function fetchPushMsg(btn1, json) {
    fetch("/time",{
 　     method: "POST",
　　    headers: {
　　　　    'Content-Type': 'application/json'
　　    },
　　    body: json
　　}).then(function(response) {
        return response.text();
    }).then(function(html) {
        var resultdiv = document.getElementById("result");
        resultdiv.className = "subgroupbox-editor";
        resultdiv.innerText += "\n" + html;
        btn1.style.transition = "opacity 0.06s";
        btn1.style.opacity = "1";
        btn1.removeAttribute("disabled");
        document.getElementById("removeOldestResultBtn").style.transform = "scale(1)";
    })
}
var detailHtml = "";
function fetchDetail() {
    fetch("/detail").then(function(response) {
        return response.text();
    }).then(function(html) {
        window.detailHtml = html;
    })
}
function removeOldestResult() {
    var resultbox = document.getElementById("resultbox");
    if (resultbox.children.length == 0) {
        document.getElementById("removeOldestResultBtn").style.transform = "scale(0)";
        return;
    }
    var oldrestResultDiv = resultbox.children[resultbox.children.length - 1];
    oldrestResultDiv.style.height = oldrestResultDiv.clientHeight + "px";
    setTimeout(function(){ oldrestResultDiv.style.height = "0px"; }, 10);
    setTimeout(function(){ resultbox.removeChild(oldrestResultDiv); }, 300);
    if (document.getElementById("resultbox").children.length == 1) {
        document.getElementById("removeOldestResultBtn").style.transform = "scale(0)";
    }
}
function subgroupbox_editor_onkeydown(e) {
    // e.metaKey for mac
    if (e.ctrlKey || e.metaKey) {
        switch (e.keyCode) {
            case 66: //ctrl+B or ctrl+b
            case 98: 
            case 73: //ctrl+I or ctrl+i
            case 105: 
            case 85: //ctrl+U or ctrl+u
            case 117: {
                e.preventDefault();    
                break;
            }
        }
    }
}
function subgroupbox_editor_onpaste(e) {
    e.preventDefault();
    var text = null;

    if (window.clipboardData && clipboardData.setData) {
        // IE
        text = window.clipboardData.getData('text');
    } else {
        text = (e.originalEvent || e).clipboardData.getData('text/plain');
    }
    if (document.body.createTextRange) {    
        if (document.selection) {
            textRange = document.selection.createRange();
        } else if (window.getSelection) {
            sel = window.getSelection();
            var range = sel.getRangeAt(0);
            
            // 创建临时元素，使得TextRange可以移动到正确的位置
            var tempEl = document.createElement("span");
            tempEl.innerHTML = "&#FEFF;";
            range.deleteContents();
            range.insertNode(tempEl);
            textRange = document.body.createTextRange();
            textRange.moveToElementText(tempEl);
            tempEl.parentNode.removeChild(tempEl);
        }
        textRange.text = text;
        textRange.collapse(false);
        textRange.select();
    } else {
        // Chrome之类浏览器
        document.execCommand("insertText", false, text);
    }
}
</script>
<form action="/webhook" name="form1" method="post" onsubmit="return homecontinue()" class="form">
<input type="text" id="webhookvalue" name="webhookvalue" value="" autocomplete="off" placeholder="群机器人的Webhook（支持钉钉、飞书）＊" autofocus="autofocus" class="txt" />
<div style="height:3px;"><textarea style="display:none;" id="webhookvaluehideTemplate">
<h4 id="webhookvaluehide" class="txt-hide">Webhook：※※※<span id="botType"></span>※※※<span onclick="backtohome()" style="margin-left:12px;"><svg version="1.1" class="larkc-svg-icon" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px" viewBox="0 0 477.873 477.873" style="enable-background:new 0 0 477.873 477.873;" xml:space="preserve">
		<path d="M392.533,238.937c-9.426,0-17.067,7.641-17.067,17.067V426.67c0,9.426-7.641,17.067-17.067,17.067H51.2
			c-9.426,0-17.067-7.641-17.067-17.067V85.337c0-9.426,7.641-17.067,17.067-17.067H256c9.426,0,17.067-7.641,17.067-17.067
			S265.426,34.137,256,34.137H51.2C22.923,34.137,0,57.06,0,85.337V426.67c0,28.277,22.923,51.2,51.2,51.2h307.2
			c28.277,0,51.2-22.923,51.2-51.2V256.003C409.6,246.578,401.959,238.937,392.533,238.937z"/><path d="M458.742,19.142c-12.254-12.256-28.875-19.14-46.206-19.138c-17.341-0.05-33.979,6.846-46.199,19.149L141.534,243.937
			c-1.865,1.879-3.272,4.163-4.113,6.673l-34.133,102.4c-2.979,8.943,1.856,18.607,10.799,21.585
			c1.735,0.578,3.552,0.873,5.38,0.875c1.832-0.003,3.653-0.297,5.393-0.87l102.4-34.133c2.515-0.84,4.8-2.254,6.673-4.13
			l224.802-224.802C484.25,86.023,484.253,44.657,458.742,19.142z M434.603,87.419L212.736,309.286l-66.287,22.135l22.067-66.202
			L390.468,43.353c12.202-12.178,31.967-12.158,44.145,0.044c5.817,5.829,9.095,13.72,9.12,21.955
			C443.754,73.631,440.467,81.575,434.603,87.419z"/></svg></span></h4></textarea></div>
<input type="text" id="secretvalue" name="secretvalue" value="" autocomplete="off" placeholder="群机器人的Secret ＊" class="txt" />
<div><textarea style="display:none;" id="secretvaluehideTemplate">
<h4 id="secretvaluehide" class="txt-hide"><span style="letter-spacing: 3px;">Secret</span>：※※※※※※</h4>
</textarea>
</div>

<div>
<textarea style="display:none;" id="detailHTMLTemplate">
<div id="detail">
<div id="subgroups"></div>
<div class="toolbar-item">
<div id="addNewSubGroupBtn" onclick='addNewSubGroup("张三\n李四\n王五")'>
<svg class="larkc-svg-icon" viewBox="0 0 1024 1024"><path d="M469.333333 469.333333v-149.333333a21.333333 21.333333 0 0 1 21.333334-21.333333h42.666666a21.333333 21.333333 0 0 1 21.333334 21.333333v149.333333h149.333333a21.333333 21.333333 0 0 1 21.333333 21.333334v42.666666a21.333333 21.333333 0 0 1-21.333333 21.333334h-149.333333v149.333333a21.333333 21.333333 0 0 1-21.333334 21.333333h-42.666666a21.333333 21.333333 0 0 1-21.333334-21.333333v-149.333333h-149.333333a21.333333 21.333333 0 0 1-21.333333-21.333334v-42.666666a21.333333 21.333333 0 0 1 21.333333-21.333334h149.333333z m42.666667 426.666667c212.074667 0 384-171.925333 384-384S724.074667 128 512 128 128 299.925333 128 512s171.925333 384 384 384z m0 85.333333C252.8 981.333333 42.666667 771.2 42.666667 512S252.8 42.666667 512 42.666667s469.333333 210.133333 469.333333 469.333333-210.133333 469.333333-469.333333 469.333333z"></path></svg>
</div>
<div id="removeNewestSubGroupBtn" onclick="removeNewestSubGroup()">
<svg class="larkc-svg-icon" viewBox="0 0 1024 1024" style="transform:rotate(45deg);"><path d="M469.333333 469.333333v-149.333333a21.333333 21.333333 0 0 1 21.333334-21.333333h42.666666a21.333333 21.333333 0 0 1 21.333334 21.333333v149.333333h149.333333a21.333333 21.333333 0 0 1 21.333333 21.333334v42.666666a21.333333 21.333333 0 0 1-21.333333 21.333334h-149.333333v149.333333a21.333333 21.333333 0 0 1-21.333334 21.333333h-42.666666a21.333333 21.333333 0 0 1-21.333334-21.333333v-149.333333h-149.333333a21.333333 21.333333 0 0 1-21.333333-21.333334v-42.666666a21.333333 21.333333 0 0 1 21.333333-21.333334h149.333333z m42.666667 426.666667c212.074667 0 384-171.925333 384-384S724.074667 128 512 128 128 299.925333 128 512s171.925333 384 384 384z m0 85.333333C252.8 981.333333 42.666667 771.2 42.666667 512S252.8 42.666667 512 42.666667s469.333333 210.133333 469.333333 469.333333-210.133333 469.333333-469.333333 469.333333z"></path></svg>
</div>
</div>

<h4>推送模版</h4>
<div class="subgroupbox">
  <div class="subgroupbox-container">
    <div class="subgroupbox-wrap">
        <pre id="subgroupbox-editor-sendcontent" spellcheck="false" class="subgroupbox-editor" contenteditable="true" onpaste="subgroupbox_editor_onpaste(event)" onkeydown="subgroupbox_editor_onkeydown(event)">今日饭搭子{number}组
组织者：{1 from SubGroup A}
参与者：{2 from SubGroup A}

...</pre>
    </div>
  </div>
</div>
</div>
</textarea>
</div>
<input type="button" class="btn1" id="btn1" value="继续" onclick="homecontinue()" />
<div id="resultbox"></div><div class="toolbar-item" style="margin-bottom: 150px;">
<div id="removeOldestResultBtn" onclick="removeOldestResult()">
<svg class="larkc-svg-icon" viewBox="0 0 1024 1024" style="transform:rotate(45deg);"><path d="M469.333333 469.333333v-149.333333a21.333333 21.333333 0 0 1 21.333334-21.333333h42.666666a21.333333 21.333333 0 0 1 21.333334 21.333333v149.333333h149.333333a21.333333 21.333333 0 0 1 21.333333 21.333334v42.666666a21.333333 21.333333 0 0 1-21.333333 21.333334h-149.333333v149.333333a21.333333 21.333333 0 0 1-21.333334 21.333333h-42.666666a21.333333 21.333333 0 0 1-21.333334-21.333333v-149.333333h-149.333333a21.333333 21.333333 0 0 1-21.333333-21.333334v-42.666666a21.333333 21.333333 0 0 1 21.333333-21.333334h149.333333z m42.666667 426.666667c212.074667 0 384-171.925333 384-384S724.074667 128 512 128 128 299.925333 128 512s171.925333 384 384 384z m0 85.333333C252.8 981.333333 42.666667 771.2 42.666667 512S252.8 42.666667 512 42.666667s469.333333 210.133333 469.333333 469.333333-210.133333 469.333333-469.333333 469.333333z"></path></svg>
</div>
</div>


</form>
<style>
::-webkit-scrollbar {
  height: 16px;
  overflow: visible;
  width: 16px;
}
::-webkit-scrollbar-thumb {
	background-color : rgba(0,0,0,.2);
	background-clip : padding-box;
	border : solid transparent;
	border-width : 1px 1px 1px 6px;
	min-height : 28px;
	padding : 100px 0 0;
	box-shadow : inset 1px 1px 0 rgba(0,0,0,.1),inset 0 -1px 0 rgba(0,0,0,.07);
}
::-webkit-scrollbar-button {
  height: 0;
  width: 0;
}

::-webkit-scrollbar-thumb:horizontal {
	border-width : 6px 1px 1px;
	padding : 0 0 0 100px;
	box-shadow : inset 1px 1px 0 rgba(0,0,0,.1),inset -1px 0 0 rgba(0,0,0,.07);
}

::-webkit-scrollbar-thumb:hover {
	background-color : rgba(0,0,0,.4);
	box-shadow : inset 1px 1px 1px rgba(0,0,0,.25);
}

::-webkit-scrollbar-thumb:active {
	background-color : rgba(0,0,0,0.5);
	box-shadow : inset 1px 1px 3px rgba(0,0,0,0.35);
}

::-webkit-scrollbar-track {
  background-clip: padding-box;
  border: solid transparent;
  border-width: 0 0 0 4px;
}
::-webkit-scrollbar-corner {
	background : transparent;
}
</style>
<script>
//fetchDetail()
</script>
`))
            case "POST":
                // 被POST, 有webhook就跳转到详情页
                req.ParseForm()
                webhookurl := req.Form.Get("webhookvalue")
                if webhookurl == "" {
                    w.WriteHeader(http.StatusNotImplemented)
                    w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
                    
                    return
                }
                
                // hasher := md5.New()
                // hasher.Write([]byte(fmt.Sprintf("%d\n%s", time.Now().UnixNano() / 1e6, webhookurl)))
                // id := hex.EncodeToString(hasher.Sum(nil))
                
                id := strings.Replace(strings.Replace(Utils.HmacSha256(fmt.Sprintf("%d\n%s", time.Now().UnixNano() / 1e6, webhookurl), "a secret"), "+", "_", -1), "/", "-", -1)
                Memcache.Store(id, &CacheData{
                                        Ip: Utils.GetIP(req),
                                        TimeAdded: time.Now().Unix(),
                                    })
                
                http.Redirect(w, req, "/loadwebhook?id=" + id, http.StatusFound)
            default:
                w.WriteHeader(http.StatusNotImplemented)
                w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
        }
    }
}

// 详情页:    1. 可点加载档案   2. 可编辑subgroup   3. 可编辑机器人推送的内容   4. 可点保存档案   5. 可点推送
// router: "/detail"
func detailHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        //time.Sleep(time.Duration(3) * time.Second)
        switch req.Method {
            case "GET":
                // 返回详情html, 由首页的ajax加载
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                w.Write([]byte(`haha
`))
            default:
                w.WriteHeader(http.StatusNotImplemented)
                w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
        }
    }
}

// 详情页   1. 编辑subgroup   2. 编辑机器人推送的内容    3. 保存     4. 推送
func loadwebhookHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        query := req.URL.Query()
        id := query.Get("id")
        justchecksession := query.Get("justchecksession")
        if id == "" {
            if justchecksession != "" {
                w.Write([]byte("notok 1"))
                return
            }
            http.Redirect(w, req, "/webhook", http.StatusFound)
            return
        }
        
        value, exists := Memcache.Load(id)
        if !exists {
            if justchecksession != "" {
                w.Write([]byte("notok 2"))
                return
            }
            http.Redirect(w, req, "/webhook", http.StatusFound)
            return
        }
        cacheData := value.(*CacheData)
        
        nowIp := Utils.GetIP(req)
        if cacheData.Ip != nowIp {
            if justchecksession != "" {
                w.Write([]byte("notok 3"))
                return
            }
            w.Write([]byte("ip not match!!!"))
            return
        }
        
        cacheData.TimeAdded = time.Now().Unix()
        Memcache.Store(id, cacheData)
        
        if justchecksession != "" {
            w.Write([]byte("ok"))
            return
        }
               
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        fmt.Fprintf(w, `<title>设置详细信息</title>
<meta content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=yes" name="viewport" />
<script>
function checksession() {
    fetch(window.location.href + "&justchecksession=1").then(function(response) {
        return response.text()
    }).then(function(data) {
        if (data.indexOf("notok") == 0) {
            alert("登录过期, 返回主页")
            return
        }
        setTimeout(function(){checksession();}, 20000)
    })
}
setTimeout(function(){checksession();}, 20000)
</script>
<body style="text-align:center">
<br><br><br>
<style>*{margin:0;padding:0} @media screen and (max-width:1981px){#logo{font-size:12px;zoom:0.5;-moz-transform:scale(0.5);-moz-transform-origin:left bottom;}}</style>
<pre id="logo" contenteditable="true">
╋╋╋╋╋╋╋╋╋┏┓╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋┏┓╋╋╋╋╋╋╋┏┓╋╋╋╋╋╋╋╋╋╋
╋╋╋╋╋╋╋╋╋┃┃╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋┏┛┗┓╋╋╋╋╋╋┃┃╋╋╋╋╋╋╋╋╋╋
╋┏━━┓┏┓┏┓┃┗━┓┏━━┓┏━┓┏━━┓┏┓┏┓┏━━┓╋╋┗┓┏┛┏━━┓┏━┛┃┏━━┓┏┓╋┏┓╋
╋┃━━┫┃┃┃┃┃┏┓┃┃┏┓┃┃┏┛┃┏┓┃┃┃┃┃┃┏┓┃╋╋╋┃┃╋┃┏┓┃┃┏┓┃┃┏┓┃┃┃╋┃┃╋
╋┣━━┃┃┗┛┃┃┗┛┃┃┗┛┃┃┃╋┃┗┛┃┃┗┛┃┃┗┛┃┏┓╋┃┗┓┃┗┛┃┃┗┛┃┃┏┓┃┃┗━┛┃╋
╋┗━━┛┗━━┛┗━━┛┗━┓┃┗┛╋┗━━┛┗━━┛┃┏━┛┗┛╋┗━┛┗━━┛┗━━┛┗┛┗┛┗━┓┏┛╋
╋╋╋╋╋╋╋╋╋╋╋╋╋┏━┛┃╋╋╋╋╋╋╋╋╋╋╋┃┃╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋┏━┛┃╋╋
╋╋╋╋╋╋╋╋╋╋╋╋╋┗━━┛╋╋╋╋╋╋╋╋╋╋╋┗┛╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋╋┗━━┛╋╋
</pre>
<br>
<h5>设置详细信息</h5>` + cacheData.Ip +`
`)
    }
}

func sendmsgHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        if err := req.ParseForm(); err != nil {
            fmt.Fprintf(w, "ParseForm() err: %v", err)
            return
        }

        webhookurl := req.Form.Get("webhookvalue")
        secretvalue := req.Form.Get("secretvalue")
        
        if webhookurl == "" && secretvalue == "" {
            http.Redirect(w, req, "/webhook", http.StatusFound)
            return
        }
        
        matched, chatApp := GetChatApp(webhookurl, secretvalue)
        var sendResult bool
        var errMsg string
        if matched {
            sendResult, errMsg = chatApp.SendMsg(time.Now().Format(time.RFC1123))
        } else {
            sendResult, errMsg = false, "not match"
        }
        
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        fmt.Fprintf(w, strconv.FormatBool(sendResult))
        fmt.Fprintf(w, " ")
        fmt.Fprintf(w, errMsg)
    }
}


// https://blog.golang.org/context/userip/userip.go
type Util struct {
	
}

func (this *Util) GenerateContent(repeat bool, number int, sendcontentArr, subGroups_A, subGroups_B, subGroups_C, subGroups_D []string) (sendContent string) {
    var appearGroupA = false
    for _, line := range sendcontentArr {
        line = strings.Replace(line, "{number}", strconv.Itoa(number), -1)
        
        matchArr := regexp.MustCompile(`\{(\-?\d+) from SubGroup ([ABCD])\}`).FindAllStringSubmatch(line, -1)
        fmt.Println(matchArr)
        
        for _, match := range matchArr {
        
            fmt.Println(match[1])
            fmt.Println(match[2])
            
            var shiftFromGroup *[]string
            
            switch match[2] {
                case "A":
                    appearGroupA = true
                    shiftFromGroup = &subGroups_A
                case "B":
                    shiftFromGroup = &subGroups_B
                case "C":
                    shiftFromGroup = &subGroups_C
                case "D":
                    shiftFromGroup = &subGroups_D
            }
            
            var replaceGroupValue string
            
            var lenShiftFromGroup = len(*shiftFromGroup)
            if lenShiftFromGroup == 0 {
                replaceGroupValue = ""
                line = strings.Replace(line, match[0], replaceGroupValue, 1)
                continue
            }
            
            randomCount, err := strconv.Atoi(match[1])
            if err != nil {
                replaceGroupValue = ""
                line = strings.Replace(line, match[0], replaceGroupValue, 1)
                continue
            }
            if randomCount > lenShiftFromGroup {
                randomCount = lenShiftFromGroup
            }
            if randomCount <= 0 {
                replaceGroupValue = ""
                line = strings.Replace(line, match[0], replaceGroupValue, 1)
                
                if match[2] == "A" {
                    appearGroupA = false
                }
                
                continue
            }
            
            randomGroupElement := (*shiftFromGroup)[0:randomCount]
            *shiftFromGroup = (*shiftFromGroup)[randomCount:]
            fmt.Println(*shiftFromGroup)
            fmt.Println(subGroups_A)
            
            replaceGroupValue = strings.Join(randomGroupElement, "、")
            line = strings.Replace(line, match[0], replaceGroupValue, 1)
        }
        
        sendContent += line + "\n"
    }
    
    if number < 10 && repeat == true && appearGroupA == true {
        if len(subGroups_A) > 0 {
            sendContent += Utils.GenerateContent(repeat, number + 1, sendcontentArr, subGroups_A, subGroups_B, subGroups_C, subGroups_D)
        }
    }
    
    return
}

func (this *Util) GetIP(req *http.Request) string {
    ip, _, err := net.SplitHostPort(req.RemoteAddr)
    if err != nil {
        return "0"
    }

    userIP := net.ParseIP(ip)
    if userIP == nil {
        //fmt.Println("userip: %q is not IP:port", req.RemoteAddr)
        return "0"
    }

    // This will only be defined when site is accessed via non-anonymous proxy
    // and takes precedence over RemoteAddr
    // Header.Get is case-insensitive
    //forward := req.Header.Get("X-Forwarded-For")

    //fmt.Println("<p>IP: %s</p>", ip)
    //fmt.Println("<p>Port: %s</p>", port)
    //fmt.Println("<p>Forwarded for: %s</p>", forward)
    
    return ip
}
func (this *Util) HmacSha256(stringToSign string, Secretvalue string) string {
	h := hmac.New(sha256.New, []byte(Secretvalue))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func GetChatApp(webhookurl, secretvalue string) (ret bool, chatApp ChatAppInterface) {
    if len(regexp.MustCompile(DingdingWebhookPatten).FindIndex([]byte(webhookurl))) == 2 {
        ret = true
        chatApp = &Dingding{Webhookurl: webhookurl, Secretvalue: secretvalue}
        
        return
    }
    if len(regexp.MustCompile(FeishuWebhookPatten).FindIndex([]byte(webhookurl))) == 2 {
        ret = true
        chatApp = &Feishu{Webhookurl: webhookurl, Secretvalue: secretvalue}
        
        return
    }
    
    ret = false
    return // regexp.FindStringSubmatch
}

type ChatAppInterface interface {
    SendMsg(msg string) (bool, string)
}
/************************************************** 钉钉机器人发送 **************************************************/
type Dingding struct {
	Webhookurl string
    Secretvalue string
}
func (this *Dingding) SendMsg(msg string) (bool, string) {
    content, data := make(map[string]string), make(map[string]interface{})
	content["content"] = msg
	data["msgtype"] = "text"
	data["text"] = content
	b, _ := json.Marshal(data)
    
    url := this.signUrl()
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
    
	if err != nil {
        //fmt.Fprintf(w, "Fatal")
        //fmt.Fprintf(w, err.Error()) // Post "https://oapi.dingtalk.com/robot/send?...": dial tcp: lookup oapi.dingtalk.com: no such host
		//log.Fatal(err)
        return false, "network error"
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
    
    return this.isSendSuccess(string(body))
}
func (this *Dingding) signUrl() (signedUrl string) {
    timestamp := time.Now().UnixNano() / 1e6
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, this.Secretvalue)
	sign := Utils.HmacSha256(stringToSign, this.Secretvalue)
	signedUrl = fmt.Sprintf("%s&timestamp=%d&sign=%s", this.Webhookurl, timestamp, sign)

    return
}
func (this *Dingding) isSendSuccess(body string) (bool, string) {
    m := make(map[string]interface{}) // {"errcode":0,"errmsg":"ok"} {"errcode":300001,"errmsg":"token is not exist"} {"errcode":310000,"errmsg":"sign not match, more: [https://ding-doc.dingtalk.com/doc#/serverapi2/qf2nxq]"}
    err := json.Unmarshal([]byte(body), &m)
    if err != nil {
        return false, err.Error()
    }
    if m["errcode"].(float64) != 0 {
        return false, m["errmsg"].(string)
    }
    
    return true, "success"
}
/************************************************** 钉钉机器人发送 **************************************************/

/************************************************** 飞书机器人发送 **************************************************/
type Feishu struct {
	Webhookurl string
    Secretvalue string
    Timestamp string
}
func (this *Feishu) SendMsg(msg string) (bool, string) {
    content, data := make(map[string]string), make(map[string]interface{})
	content["text"] = msg
	timestamp := time.Now().Unix()
	data["timestamp"] = timestamp
	data["sign"] = this.getSign(timestamp)
	data["msg_type"] = "text"
	data["content"] = content
	b, _ := json.Marshal(data)
    
    url := this.Webhookurl
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
    
	if err != nil {
		//log.Fatal(err)
        return false, "network error"
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
    
    return this.isSendSuccess(string(body))
}
func (this *Feishu) getSign(timestamp int64) string { // 摘自官方
   stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + this.Secretvalue
   var data []byte
   h := hmac.New(sha256.New, []byte(stringToSign))
   _, err := h.Write(data)
   if err != nil {
      return ""
   }
   signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
   
   return signature
}
func (this *Feishu) isSendSuccess(body string) (bool, string) {
    m := make(map[string]interface{}) // {"Extra":null,"StatusCode":0,"StatusMessage":"success"} {"code":19001,"msg":"param invalid: incoming webhook access token invalid"} {"code":19021,"msg":"sign match fail or timestamp is not within one hour from current time"}
    err := json.Unmarshal([]byte(body), &m)
    if err != nil {
        return false, err.Error()
    }
    
    if _, isset := m["StatusCode"]; isset {
        if m["StatusCode"].(float64) == 0 {
            return true, "success"
        }
        return false, strconv.FormatFloat(m["StatusCode"].(float64), 'g', 6, 64)
    }
    return false, m["msg"].(string)
}
/************************************************** 飞书机器人发送 **************************************************/