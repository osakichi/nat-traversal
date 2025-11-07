package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// main はNATトラバーサルのピアアプリケーションを起動します。
// ブローカーに登録後、リモートピアと直接P2P通信を確立します。
func main() {
	// ローカルでリッスンするアドレスとポート
	host := "0.0.0.0:65432"
	// このピアの名前
	localName := "peerA"
	// 接続先のリモートピアの名前
	remoteName := "peerZ"
	// ブローカーサーバーのアドレス
	broker := "prgmr.nohohon.jp:65432"

	// ローカルアドレスの解決
	hostAddr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		log.Fatal("net.ResolveUDPAddr(host): ", err)
		os.Exit(1)
	}

	// ブローカーアドレスの解決
	brokerAddr, err := net.ResolveUDPAddr("udp", broker)
	if err != nil {
		log.Fatal("net.ResolveUDPAddr(broker): ", err)
		os.Exit(3)
	}

	// UDP接続のリッスン開始
	conn, err := net.ListenUDP("udp", hostAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// ブローカーに自身を登録
	err = regLocal(conn, localName, brokerAddr)
	if err != nil {
		log.Fatal("regLocal(", localName, "): ", err)
		os.Exit(2)
	}

	// NATバインディングが確立するまで待機
	// UDPホールパンチングを成功させるために必要
	time.Sleep(10 * time.Second)

	// ブローカーからリモートピアのアドレスを取得
	remoteAddr, err := getRemote(conn, remoteName, brokerAddr)
	if err != nil {
		log.Fatal("regLocal(", remoteName, "): ", err)
		os.Exit(2)
	}

	// メッセージ受信用のゴルーチンを起動
	go server(conn)
	// メッセージ送信用のゴルーチンを起動
	go client(conn, remoteAddr, localName)

	// メインスレッドは無限ループで待機
	for {
		time.Sleep(10 * time.Second)
	}
}

// server はUDPメッセージを継続的に受信するゴルーチンです。
// リモートピアから送られてくるメッセージをログに出力します。
func server(conn *net.UDPConn) {
	// UDPパケット受信用バッファ（最大4096バイト）
	buf := make([]byte, 4096)
	for {
		// UDPパケットを受信
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		// 受信したメッセージをログ出力
		log.Print("Recv ", remoteAddr, " : ", string(buf[:n]))
	}
}

// client は定期的にリモートピアへメッセージを送信するゴルーチンです。
// 5秒間隔でカウンター付きのメッセージを送信し、P2P接続を維持します。
func client(conn *net.UDPConn, remoteAddr *net.UDPAddr, name string) {
	// メッセージのカウンター
	n := 0
	for {
		// "<ピア名> <カウンター>" 形式のメッセージを作成
		msg := fmt.Sprintf("%s %d", name, n)
		// リモートピアへメッセージを送信
		_, err := conn.WriteToUDP([]byte(msg), remoteAddr)
		if err != nil {
			log.Fatal(err)
		}
		log.Print("Send ", remoteAddr, " : ", msg)

		// カウンターをインクリメント
		n++

		// 5秒待機
		time.Sleep(5 * time.Second)
	}
}

// regLocal はブローカーに自身のピア情報を登録します。
// ピア名とローカルアドレスをREGコマンドでブローカーに送信し、
// ブローカーが認識したこのピアのパブリックアドレスを受け取ります。
func regLocal(conn *net.UDPConn, name string, brokerAddr *net.UDPAddr) error {
	// "REG <ピア名> <ローカルアドレス>" 形式のメッセージを作成
	msg := fmt.Sprintf("REG %s %s", name, conn.LocalAddr().String())
	// ブローカーへREGコマンドを送信
	_, err := conn.WriteToUDP([]byte(msg), brokerAddr)
	if err != nil {
		log.Fatal(err)
	}

	// ブローカーからのレスポンスを受信
	buf := make([]byte, 4096)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		log.Fatal(err)
	}

	// レスポンスを解析
	col := strings.Split(string(buf[:n]), " ")
	if col[0] != "OK" {
		log.Fatal("REG failed : ", name)
	}

	// 登録成功をログ出力（パブリックアドレスを含む）
	log.Print("REG success : ", name, " = ", col[1])

	return nil
}

// getRemote はブローカーから指定されたピアのアドレスを取得します。
// GETコマンドでリモートピアの名前を問い合わせ、
// ブローカーに登録されているそのピアのパブリックアドレスを取得します。
func getRemote(conn *net.UDPConn, name string, brokerAddr *net.UDPAddr) (*net.UDPAddr, error) {
	// "GET <ピア名>" 形式のメッセージを作成
	msg := fmt.Sprintf("GET %s", name)
	// ブローカーへGETコマンドを送信
	_, err := conn.WriteToUDP([]byte(msg), brokerAddr)
	if err != nil {
		log.Fatal(err)
	}

	// ブローカーからのレスポンスを受信
	buf := make([]byte, 4096)
	n, remoteAddr, err := conn.ReadFromUDP(buf)
	if err != nil {
		log.Fatal(err)
	}

	// レスポンスを解析
	col := strings.Split(string(buf[:n]), " ")
	if col[0] != "OK" {
		log.Fatal("GET failed : ", name)
	}

	// 取得成功をログ出力（リモートピアのアドレスを含む）
	log.Print("GET success : ", name, " = ", col[1])

	// 文字列形式のアドレスをUDPAddr構造体に変換
	remoteAddr, err = net.ResolveUDPAddr("udp", col[1])
	if err != nil {
		log.Fatal("net.ResolveUDPAddr(regLocal): ", err)
	}

	return remoteAddr, nil
}
