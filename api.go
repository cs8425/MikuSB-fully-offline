package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	ServerName = "MikuSB"
)

// --- Middleware ---
func reqlog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Vln(3, r.Method, r.URL, r.RemoteAddr, r.Host)
		if *Verbosity >= 6 {
			for i, hdr := range r.Header {
				Vln(6, "---", i, len(hdr), hdr)
			}
			Vln(6, "")
		}
		next.ServeHTTP(w, r)
	})
}

func initRoute(mux *http.ServeMux, mockServerAddr string) {
	// Register explicit routes

	// :31443, collect log?
	// POST /data/report/v3
	mux.HandleFunc("/data/report/v3", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"message":"success"}`))
	})

	// :18443
	// POST /getGameConfig
	mux.HandleFunc("/getGameConfig", getGameConfig)

	// :13443
	mux.HandleFunc("/seasun/config", getSeasunConfig)
	mux.HandleFunc("/seasun/loginByToken", loginByToken)
	mux.HandleFunc("/seasun/login", loginByToken) // return same still work
	mux.HandleFunc("/seasun/getAccountInfoForGame", getAccountInfoForGame)
	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		getQuery(w, r, mockServerAddr)
	})
	// mux.HandleFunc("/query_version", getQuery)
	// mux.HandleFunc("/query_version={version}", getQuery)

	mux.HandleFunc("/api/serverlist", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, buildServerList("", mockServerAddr))
	})
	mux.HandleFunc("/account/query-uid/", queryUid) // Prefix match, `/account/query-uid/{appId}`
	mux.HandleFunc("/health", getHealth)
	mux.HandleFunc("/bisdk/batchpush", getBatchPush)

	mux.HandleFunc("/api/auth/guest", authGuest)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/query_version":
			version := r.URL.Query().Get("version")
			writeJSON(w, buildServerList(version, mockServerAddr))
			return
		case strings.HasPrefix(path, "/query_version="):
			version := strings.TrimPrefix(r.URL.Path, "/query_version=")
			writeJSON(w, buildServerList(version, mockServerAddr))
			return
		default:
		}

		// Fallback/ServerList Logic (The "MapFallback" part)
		combined := strings.ToLower(path + "?" + r.URL.RawQuery)
		isServerList := strings.Contains(combined, "server") ||
			strings.Contains(combined, "version") ||
			strings.Contains(combined, "query_version") ||
			strings.Contains(combined, "serverlist")

		if isServerList {
			// Extract version if present in query or path
			version := r.URL.Query().Get("version")
			// if path is /query_version=1.0, version is "1.0"
			if after, ok := strings.CutPrefix(path, "/query_version="); ok {
				version = after
			}
			writeJSON(w, buildServerList(version, mockServerAddr))
			return
		}
		// Final Fallback
		rsp := map[string]any{
			"code":    0,
			"message": "ok",
			"service": ServerName,
			"path":    path,
			"query":   r.URL.RawQuery,
		}
		writeJSON(w, rsp)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func getGameConfig(w http.ResponseWriter, r *http.Request) {
	a := struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data any    `json:"data"`
	}{
		Code: "0",
		Msg:  "success",
		Data: struct {
			AgreementUpdateTime      string   `json:"agreementUpdateTime"`
			QqGroup                  *string  `json:"qqGroup"`
			AppDownLoadUrl           string   `json:"appDownLoadUrl"`
			OpenActivationCode       bool     `json:"openActivationCode"`
			LoginType                []string `json:"loginType"`
			EnableReportDataToDouyin bool     `json:"enableReportDataToDouyin"`
		}{
			AgreementUpdateTime:      "1728552600000",
			QqGroup:                  nil,
			AppDownLoadUrl:           "",
			OpenActivationCode:       false,
			LoginType:                []string{"channel"},
			EnableReportDataToDouyin: false,
		},
	}
	writeJSON(w, a)
}

func getSeasunConfig(w http.ResponseWriter, r *http.Request) {
	a := struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data any    `json:"data"`
	}{
		Code: 0,
		Msg:  "操作成功",
		Data: struct {
			PlatformPrivacyAgreement string   `json:"platformPrivacyAgreement"`
			PayType                  []string `json:"payType"`
			LoginType                []string `json:"loginType"`
			CloseGeetest             bool     `json:"closeGeetest"`
			UserAgreement            string   `json:"userAgreement"`
			PrivacyAgreement         string   `json:"privacyAgreement"`
			InitPrivacyUpdateTime    int      `json:"initPrivacyUpdateTime"`
			PlatformUserAgreement    string   `json:"platformUserAgreement"`
			AccountPublicKey         string   `json:"accountPublicKey,omitempty"`
			PayChannel               []string `json:"payChannel,omitempty"`
			RegisterPrivacyUrl       string   `json:"registerPrivacyUrl"`
			LoginPrivacyUrl          string   `json:"loginPrivacyUrl"`
		}{
			PlatformPrivacyAgreement: "https://www.amazingseasun.com/privacy.html?lang=zh-Hant&gamecode=200001086",
			PayType:                  []string{"mycard"},
			LoginType: []string{
				"mail",
				"google",
				"twitter",
				"guest",
				"steam",
			},
			CloseGeetest:          false,
			UserAgreement:         "https://www.amazingseasun.com/user.html?lang=zh-Hant&gamecode=111111680",
			PrivacyAgreement:      "https://www.amazingseasun.com/privacy.html?lang=zh-Hant&gamecode=111111680",
			InitPrivacyUpdateTime: 0,
			PlatformUserAgreement: "https://www.amazingseasun.com/user.html?lang=zh-Hant&gamecode=200001086",
			AccountPublicKey:      "",  // TODO: pem string?
			PayChannel:            nil, // TODO: struct
			RegisterPrivacyUrl:    "https://xgsdk.xoyo.games:13443/seasun/privacy-agreement/200001086/register/privacy.html?language=zh-Hant",
			LoginPrivacyUrl:       "https://xgsdk.xoyo.games:13443/seasun/privacy-agreement/111111680/login/privacy.html?language=zh-Hant",
		},
	}
	writeJSON(w, a)
}

func loginByToken(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	uidStr := r.URL.Query().Get("uid")
	if uidStr == "" {
		uidStr = r.FormValue("form_uid")
	}
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		uid = 10001
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.FormValue("form_token")
	}
	if token == "" {
		token = "mock_token_string" // Simplified for example
	}

	a := struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data any    `json:"data"`
	}{
		Code: 0,
		Msg:  "操作成功",
		Data: struct {
			AssociatedAccounts []string `json:"associatedAccounts"`
			IsFirstLogin       bool     `json:"isFirstLogin"`
			IsNeedKoreaSciAuth bool     `json:"isNeedKoreaSciAuth"`
			KsOpenId           string   `json:"ksOpenId"`
			Nickname           string   `json:"nickname"`
			PassportId         string   `json:"passportId"`
			PlayerFillAgeUrl   string   `json:"playerFillAgeUrl"`
			Status             int      `json:"status"`
			ThirdPartyUid      string   `json:"thirdPartyUid"`
			Token              string   `json:"token"`
			Type               string   `json:"type"`
			Uid                int      `json:"uid"`
		}{
			AssociatedAccounts: []string{},
			IsFirstLogin:       false,
			IsNeedKoreaSciAuth: false,
			KsOpenId:           "ks_" + uidStr,
			Nickname:           ServerName,
			PassportId:         uidStr,
			PlayerFillAgeUrl:   "",
			Status:             0,
			ThirdPartyUid:      "",
			Token:              token,
			Type:               "guest", // "google",
			Uid:                int(uid),
		},
	}
	writeJSON(w, a)
}

func getAccountInfoForGame(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	uidString := r.URL.Query().Get("uid")
	if uidString == "" {
		uidString = r.FormValue("form_uid")
	}
	if uidString == "" {
		uidString = "10001"
	}

	rsp := map[string]any{
		"code": 0,
		"data": map[string]any{
			"bindAccountTypes": []string{"google"},
			"channelUid":       uidString,
			"loginAccountType": "google",
			"nickName":         ServerName,
			"passportId":       uidString,
			"uid":              "seasun__" + uidString,
		},
		"msg": "操作成功",
	}
	writeJSON(w, rsp)
}

func getBatchPush(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"code": 0, "ret": 0, "msg": "ok", "message": "ok"})
}

func getQuery(w http.ResponseWriter, r *http.Request, mockServerAddr string) {
	host, port, _ := net.SplitHostPort(mockServerAddr)
	servers := []map[string]any{
		{
			"id":        1,
			"server_id": 1,
			"name":      ServerName,
			"host":      host,
			"port":      port,
		},
	}
	writeJSON(w, servers)
}

func queryUid(w http.ResponseWriter, r *http.Request) {
	// Path: /account/query-uid/{appId}
	parts := strings.Split(r.URL.Path, "/")
	appId := "default"
	if len(parts) > 3 {
		appId = parts[3]
	}
	_ = appId // appId used if needed, but logic uses authInfo or fallback

	authInfo := r.URL.Query().Get("authInfo")
	uid := extractUid(authInfo)
	if uid == "" {
		uid = "10001"
	}

	rsp := map[string]any{
		"code": "0",
		"msg":  "success",
		"data": map[string]any{
			"uid": "seasun__" + uid,
		},
	}
	writeJSON(w, rsp)
}

func authGuest(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("Token")
	writeJSON(w, map[string]any{
		"Provider": "Guest",
		"Token":    token,
		"Account":  "Account",
		"Pid":      "123813131321312",
	})
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"status":  "ok",
		"service": ServerName,
	})
}

func buildServerList(version string, mockServerAddr string) map[string]any {
	host, port, _ := net.SplitHostPort(mockServerAddr)
	return map[string]any{
		"code":        0,
		"ret":         0,
		"msg":         "ok",
		"message":     "ok",
		"version":     version,
		"server_time": time.Now().Unix(),
		"servers": []map[string]any{
			{
				"id":        1,
				"server_id": 1,
				"host":      host,
				"port":      port,
				"status":    1,
				"state":     1,
				"is_open":   true,
				"open":      true,
				"recommend": true,
			},
		},
		// "game_server": map[string]any{
		// 	"host": config.GameServer.PublicAddress,
		// 	"ip":   config.GameServer.PublicAddress,
		// 	"port": config.GameServer.Port,
		// },
		// "http_server": map[string]any{
		// 	"host": config.HttpServer.PublicAddress,
		// 	"port": config.HttpServer.Port,
		// },
	}
}

func extractUid(authInfo string) string {
	if authInfo == "" {
		return ""
	}
	// Handle padding for Base64
	normalized := authInfo
	for len(normalized)%4 != 0 {
		normalized += "="
	}

	decoded, err := base64.StdEncoding.DecodeString(normalized)
	if err != nil {
		return ""
	}

	// Parse JSON string
	var doc struct {
		Uid string `json:"uid"`
	}
	if err := json.Unmarshal(decoded, &doc); err != nil {
		return ""
	}
	return doc.Uid
}
