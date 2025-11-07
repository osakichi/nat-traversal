package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

// main はNATトラバーサルのブローカーサーバーを起動します。
// ピアの登録と検索を仲介するランデブーサーバーとして動作します。
func main() {
	// ブローカーがリッスンするアドレスとポート
	local := "0.0.0.0:65432"

	// UDPアドレスの解決
	localAddr, err := net.ResolveUDPAddr("udp", local)
	if err != nil {
		log.Fatal("net.ResolveUDPAddr(local): ", err)
		os.Exit(1)
	}

	// UDP接続のリッスン開始
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// ピア情報を保持するキャッシュを初期化
	// デフォルト有効期限: 5分, クリーンアップ間隔: 10分
	dataCache := cache.New(5*time.Minute, 10*time.Minute)

	// UDPパケット受信用バッファ（最大4096バイト）
	buf := make([]byte, 4096)

	// メインループ: クライアントからのリクエストを処理
	for {
		// UDPパケットを受信
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}

		// 受信メッセージをログ出力
		msg := string(buf[:n])
		log.Print("Recv ", remoteAddr, " : ", msg)

		// メッセージをスペースで分割してコマンドを解析
		col := strings.Split(msg, " ")
		if len(col) < 1 {
			continue
		}

		// コマンドに応じた処理
		switch col[0] {
		case "GET":
			// ピアの検索: 指定された名前のピアのアドレスを返す
			item, found := dataCache.Get(col[1])
			if found {
				msg = fmt.Sprintf("OK %s", item.(string))
			} else {
				// ピアが見つからない場合
				msg = "NF "
			}
		case "REG":
			// ピアの登録: ピア名とそのパブリックアドレスを保存
			remote := remoteAddr.String()
			dataCache.Set(col[1], remote, cache.NoExpiration)
			msg = fmt.Sprintf("OK %s", remote)
		default:
			// 無効なコマンド
			msg = "NG "
		}

		// レスポンスを送信
		_, err = conn.WriteToUDP([]byte(msg), remoteAddr)
		if err != nil {
			log.Fatal(err)
		}
		log.Print("Send ", remoteAddr, " : ", msg)
	}
}
