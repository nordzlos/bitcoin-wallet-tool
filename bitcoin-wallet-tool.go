package main

import (
	"bufio"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/fatih/color"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/term"
)

// Language constants
const (
	LANG_ENGLISH  = "en"
	LANG_CHINESE  = "zh"
	LANG_JAPANESE = "ja"
	LANG_RUSSIAN  = "ru"
)

// ASCII Art and UI Constants
const (
	LOGO = `
██████╗ ████████╗ ██████╗    ██╗    ██╗ █████╗ ██╗     ██╗     ███████╗████████╗
██╔══██╗╚══██╔══╝██╔════╝    ██║    ██║██╔══██╗██║     ██║     ██╔════╝╚══██╔══╝
██████╔╝   ██║   ██║         ██║ █╗ ██║███████║██║     ██║     █████╗     ██║   
██╔══██╗   ██║   ██║         ██║███╗██║██╔══██║██║     ██║     ██╔══╝     ██║   
██████╔╝   ██║   ╚██████╗    ╚███╔███╔╝██║  ██║███████╗███████╗███████╗   ██║   
╚═════╝    ╚═╝    ╚═════╝     ╚══╝╚══╝ ╚═╝  ╚═╝╚══════╝╚══════╝╚══════╝   ╚═╝   
                                                                                 
`
	MENU_HEADER_WIDTH = 80
	DISPLAY_WIDTH     = 60
)

// ANSI color formatting - limited to retro terminal colors
var (
	// Primary colors
	green     = color.New(color.FgGreen).SprintFunc()
	greenBold = color.New(color.FgGreen, color.Bold).SprintFunc()
	amber     = color.New(color.FgYellow).SprintFunc()
	white     = color.New(color.FgWhite).SprintFunc()
	whiteBold = color.New(color.FgWhite, color.Bold).SprintFunc()
	red       = color.New(color.FgRed).SprintFunc()
	cyan      = color.New(color.FgCyan).SprintFunc()
	blue      = color.New(color.FgBlue).SprintFunc()
	magenta   = color.New(color.FgMagenta).SprintFunc()
	
	// Background colors
	greenBg  = color.New(color.BgGreen, color.FgBlack).SprintFunc()
	amberBg  = color.New(color.BgYellow, color.FgBlack).SprintFunc()
	cyanBg   = color.New(color.BgCyan, color.FgBlack).SprintFunc()
	magentaBg = color.New(color.BgMagenta, color.FgBlack).SprintFunc()
)

// Mempool API endpoints
// add more if needed (works best with multiple endpoints to avoid rate limiting
var (
	mempoolEndpoints = []string{
		"mempool.space"
	}
	currentEndpointIndex = 0
	endpointMutex        sync.Mutex
	currentEndpoint      string // Track the currently active endpoint
	endpointFailures     = make(map[string]int) // Track failures per endpoint
	endpointFailureMutex sync.Mutex
)

// Global language setting
var currentLanguage = LANG_ENGLISH

// Translation maps for different languages
var translations = map[string]map[string]string{
	LANG_ENGLISH: {
		"app_title":                "BITCOIN WALLET TOOL",
		"welcome_message":          "WELCOME TO THE CRYPTO UNDERGROUND",
		"risk_warning":             "*** USE AT YOUR OWN RISK ***",
		"disconnect":               "DISCONNECT",
		"info":                     "INFO",
		"success":                  "SUCCESS",
		"warning":                  "WARNING",
		"error":                    "ERROR",
		"scan":                     "SCAN",
		"stats":                    "STATS",
		"active_wallet_found":      "ACTIVE WALLET FOUND",
		"menu_check_balance":       "CHECK WALLET BALANCE",
		"menu_view_transactions":   "VIEW TRANSACTION HISTORY",
		"menu_generate_addresses":  "GENERATE ADDRESSES FROM SEED PHRASE",
		"menu_scan_wallets":        "SCAN RANDOM WALLETS",
		"menu_multi_scan":          "MULTI-INSTANCE WALLET SCANNER",
		"menu_exit":                "EXIT",
		"enter_command":            "ENTER COMMAND >",
		"enter_address":            "Enter Bitcoin address >",
		"enter_tx_limit":           "Enter number of transactions to display (default: 5) >",
		"enter_seed_phrase":        "Enter your seed phrase >",
		"enter_address_count":      "Enter number of addresses to generate for each type (default: 5) >",
		"enter_instance_count":     "Enter number of scanner instances to spawn >",
		"no_address":               "No address provided.",
		"no_seed_phrase":           "No seed phrase provided.",
		"invalid_seed_phrase":      "Invalid seed phrase. Please check and try again.",
		"invalid_command":          "Invalid command. Please enter a number between 1 and 7.",
		"press_enter":              "Press Enter to return to main menu...",
		"connecting":               "Connecting to blockchain...",
		"retrieving_tx_data":       "Retrieving transaction data...",
		"generating_addresses":     "Generating addresses...",
		"scanning_blockchain":      "Scanning blockchain...",
		"complete":                 "COMPLETE",
		"wallet_balance":           "BITCOIN WALLET BALANCE",
		"address":                  "Address:",
		"total_balance":            "Total Balance:",
		"confirmed":                "Confirmed:",
		"unconfirmed":              "Unconfirmed:",
		"transactions":             "Transactions:",
		"calculation_details":      "Calculation Details:",
		"formula":                  "Formula:",
		"explorer":                 "Explorer:",
		"url":                      "URL:",
		"transaction":              "TRANSACTION",
		"txid":                     "TXID:",
		"status":                   "Status:",
		"block":                    "Block:",
		"time":                     "Time:",
		"total_input":              "Total Input:",
		"total_output":             "Total Output:",
		"fee":                      "Fee:",
		"size_weight":              "Size/Weight:",
		"explorer_url":             "Explorer URL:",
		"pending":                  "Pending",
		"confirmed_status":         "Confirmed",
		"wallet_has_funds":         "Wallet has funds: %.8f BTC",
		"wallet_zero_balance":      "Wallet has zero balance",
		"fetching_tx_history":      "Fetching transaction history for address: %s (limit: %d)",
		"no_transactions":          "No transactions found for address: %s",
		"found_transactions":       "Found %d transactions for address: %s",
		"generating_addresses_msg": "Generating %d addresses for each BIP standard...",
		"check_balances":           "Do you want to check balances for these addresses? (y/n) >",
		"checking_balances":        "Checking balances for generated addresses...",
		"checking_address":         "Checking %s address: %s",
		"found_funds":              "Found funds in %s address: %s",
		"balance":                  "Balance: %.8f BTC",
		"fetching_details":         "Fetching detailed information...",
		"view_transactions":        "Do you want to see transactions for this address? (y/n):",
		"no_funds_found":           "No funds found in any of the generated addresses.",
		"random_wallet_scanner":    "RANDOM WALLET SCANNER",
		"multi_scanner":            "MULTI-INSTANCE WALLET SCANNER",
		"scanner_warning":          "WARNING: This tool is for educational purposes only. Scanning for random wallets is extremely unlikely to find any funds. Do not use for illegal purposes.",
		"starting_scanner":         "Starting random wallet scanner. Results will be saved to %s",
		"starting_multi_scanner":   "Starting %d scanner instances. Press Ctrl+C to stop all scanners.",
		"instance_started":         "Instance #%d started. Results will be saved to %s",
		"press_ctrl_c":             "Press Ctrl+C to stop the scanner",
		"testing_address":          "Testing: %s (%s)",
		"wallets_checked":          "Wallets checked: %d | Active wallets found: %d",
		"stopping_scanner":         "Stopping wallet scanner...",
		"stopping_all_scanners":    "Stopping all scanner instances...",
		"total_wallets_checked":    "Total wallets checked: %d",
		"total_active_wallets":     "Total active wallets found: %d",
		"results_saved":            "Results saved to: %s",
		"seed_phrase":              "Seed Phrase:",
		"path":                     "Path:",
		"disconnecting":            "Disconnecting from the crypto underground...",
		"clearing_traces":          "Clearing traces...",
		"connection_terminated":    "Connection terminated.",
		"testing_endpoints":        "Testing Mempool API endpoints for availability...",
		"endpoint":                 "Endpoint: %s Status: %s Latency: %s",
		"found_working_endpoints":  "Found %d working Mempool API endpoints",
		"using_endpoint":           "Using endpoint: %s",
		"rotating_endpoint":        "Rotating from %s to %s due to API issues",
		"endpoint_error":           "Error with endpoint %s: %v",
		"endpoint_status_code":     "Endpoint %s returned status code %d",
		"json_parse_error":         "Failed to parse JSON from %s: %v",
		"generated_addresses":      "GENERATED BITCOIN ADDRESSES",
		"language_selection":       "LANGUAGE SELECTION",
		"select_language":          "Select your preferred language:",
		"english":                  "English",
		"chinese":                  "Chinese (中文)",
		"japanese":                 "Japanese (日本語)",
		"russian":                  "Russian (Русский)",
		"language_selected":        "Language set to %s",
		"instance_status":          "Instance #%d: %s",
		"checking_wallet":          "Checking %s via %s",
		"endpoint_removed":         "Endpoint %s removed due to consistent failures",
		"endpoint_warning":         "Warning: Endpoint %s experiencing issues",
		"instance_idle":            "Idle",
		"instance_active":          "Active",
		"instance_error":           "Error",
		"instance_success":         "Success",
	},
	LANG_CHINESE: {
		"app_title":                "比特币钱包工具 v1.0.0",
		"welcome_message":          "欢迎来到加密货币地下世界",
		"risk_warning":             "*** 使用风险自负 ***",
		"disconnect":               "断开连接",
		"info":                     "信息",
		"success":                  "成功",
		"warning":                  "警告",
		"error":                    "错误",
		"scan":                     "扫描",
		"stats":                    "统计",
		"active_wallet_found":      "发现活跃钱包",
		"menu_check_balance":       "查询钱包余额",
		"menu_view_transactions":   "查看交易历史",
		"menu_generate_addresses":  "从助记词生成地址",
		"menu_scan_wallets":        "扫描随机钱包",
		"menu_multi_scan":          "多实例钱包扫描器",
		"menu_exit":                "退出",
		"enter_command":            "输入命令 >",
		"enter_address":            "输入比特币地址 >",
		"enter_tx_limit":           "输入要显示的交易数量（默认：5）>",
		"enter_seed_phrase":        "输入您的助记词 >",
		"enter_address_count":      "输入要为每种类型生成的地址数量（默认：5）>",
		"enter_instance_count":     "输入要启动的扫描器实例数量 >",
		"no_address":               "未提供地址。",
		"no_seed_phrase":           "未提供助记词。",
		"invalid_seed_phrase":      "无效的助记词。请检查并重试。",
		"invalid_command":          "无效命令。请输入1到7之间的数字。",
		"press_enter":              "按回车键返回主菜单...",
		"connecting":               "正在连接区块链...",
		"retrieving_tx_data":       "正在检索交易数据...",
		"generating_addresses":     "正在生成地址...",
		"scanning_blockchain":      "正在扫描区块链...",
		"complete":                 "完成",
		"wallet_balance":           "比特币钱包余额",
		"address":                  "地址：",
		"total_balance":            "总余额：",
		"confirmed":                "已确认：",
		"unconfirmed":              "未确认：",
		"transactions":             "交易：",
		"calculation_details":      "计算详情：",
		"formula":                  "公式：",
		"explorer":                 "区块浏览器：",
		"url":                      "网址：",
		"transaction":              "交易",
		"txid":                     "交易ID：",
		"status":                   "状态：",
		"block":                    "区块：",
		"time":                     "时间：",
		"total_input":              "总输入：",
		"total_output":             "总输出：",
		"fee":                      "手续费：",
		"size_weight":              "大小/权重：",
		"explorer_url":             "浏览器网址：",
		"pending":                  "待处理",
		"confirmed_status":         "已确认",
		"wallet_has_funds":         "钱包有资金：%.8f BTC",
		"wallet_zero_balance":      "钱包余额为零",
		"fetching_tx_history":      "正在获取地址的交易历史：%s（限制：%d）",
		"no_transactions":          "未找到地址的交易：%s",
		"found_transactions":       "找到%d笔交易，地址：%s",
		"generating_addresses_msg": "正在为每个BIP标准生成%d个地址...",
		"check_balances":           "您想检查这些地址的余额吗？(y/n) >",
		"checking_balances":        "正在检查生成的地址余额...",
		"checking_address":         "正在检查%s地址：%s",
		"found_funds":              "在%s地址中发现资金：%s",
		"balance":                  "余额：%.8f BTC",
		"fetching_details":         "正在获取详细信息...",
		"view_transactions":        "您想查看此地址的交易吗？(y/n)：",
		"no_funds_found":           "在任何生成的地址中都未找到资金。",
		"random_wallet_scanner":    "随机钱包扫描器",
		"multi_scanner":            "多实例钱包扫描器",
		"scanner_warning":          "警告：此工具仅用于教育目的。扫描随机钱包几乎不可能找到任何资金。请勿用于非法目的。",
		"starting_scanner":         "正在启动随机钱包扫描器。结果将保存到%s",
		"starting_multi_scanner":   "正在启动%d个扫描器实例。按Ctrl+C停止所有扫描器。",
		"instance_started":         "实例#%d已启动。结果将保存到%s",
		"press_ctrl_c":             "按Ctrl+C停止扫描器",
		"testing_address":          "测试中：%s（%s）",
		"wallets_checked":          "已检查钱包：%d | 发现活跃钱包：%d",
		"stopping_scanner":         "正在停止钱包扫描器...",
		"stopping_all_scanners":    "正在停止所有扫描器实例...",
		"total_wallets_checked":    "总共检查的钱包：%d",
		"total_active_wallets":     "总共发现的活跃钱包：%d",
		"results_saved":            "结果已保存到：%s",
		"seed_phrase":              "助记词：",
		"path":                     "路径：",
		"disconnecting":            "正在断开与加密货币地下世界的连接...",
		"clearing_traces":          "正在清除痕迹...",
		"connection_terminated":    "连接已终止。",
		"testing_endpoints":        "正在测试Mempool API端点的可用性...",
		"endpoint":                 "端点：%s 状态：%s 延迟：%s",
		"found_working_endpoints":  "找到%d个工作的Mempool API端点",
		"using_endpoint":           "正在使用端点：%s",
		"rotating_endpoint":        "由于API问题，正在从%s轮换到%s",
		"endpoint_error":           "端点%s错误：%v",
		"endpoint_status_code":     "端点%s返回状态码%d",
		"json_parse_error":         "无法从%s解析JSON：%v",
		"generated_addresses":      "生成的比特币地址",
		"language_selection":       "语言选择",
		"select_language":          "选择您的首选语言：",
		"english":                  "英语 (English)",
		"chinese":                  "中文",
		"japanese":                  "日语 (日本語)",
		"russian":                  "俄语 (Русский)",
		"language_selected":        "语言设置为%s",
		"instance_status":          "实例 #%d: %s",
		"checking_wallet":          "正在通过%s检查%s",
		"endpoint_removed":         "由于持续失败，端点%s已被移除",
		"endpoint_warning":         "警告：端点%s出现问题",
		"instance_idle":            "空闲",
		"instance_active":          "活跃",
		"instance_error":           "错误",
		"instance_success":         "成功",
	},
	LANG_JAPANESE: {
		"app_title":                "ビットコインウォレットツール v1.0.0",
		"welcome_message":          "暗号通貨アンダーグラウンドへようこそ",
		"risk_warning":             "*** 自己責任でご利用ください ***",
		"disconnect":               "切断",
		"info":                     "情報",
		"success":                  "成功",
		"warning":                  "警告",
		"error":                    "エラー",
		"scan":                     "スキャン",
		"stats":                    "統計",
		"active_wallet_found":      "アクティブなウォレットが見つかりました",
		"menu_check_balance":       "ウォレット残高を確認",
		"menu_view_transactions":   "取引履歴を表示",
		"menu_generate_addresses":  "シードフレーズからアドレスを生成",
		"menu_scan_wallets":        "ランダムウォレットをスキャン",
		"menu_multi_scan":          "マルチインスタンスウォレットスキャナー",
		"menu_exit":                "終了",
		"enter_command":            "コマンドを入力 >",
		"enter_address":            "ビットコインアドレスを入力 >",
		"enter_tx_limit":           "表示する取引数を入力（デフォルト：5）>",
		"enter_seed_phrase":        "シードフレーズを入力 >",
		"enter_address_count":      "各タイプで生成するアドレス数を入力（デフォルト：5）>",
		"enter_instance_count":     "起動するスキャナーインスタンスの数を入力 >",
		"no_address":               "アドレスが提供されていません。",
		"no_seed_phrase":           "シードフレーズが提供されていません。",
		"invalid_seed_phrase":      "無効なシードフレーズです。確認して再試行してください。",
		"invalid_command":          "無効なコマンドです。1から7の間の数字を入力してください。",
		"press_enter":              "メインメニューに戻るにはEnterキーを押してください...",
		"connecting":               "ブロックチェーンに接続中...",
		"retrieving_tx_data":       "取引データを取得中...",
		"generating_addresses":     "アドレスを生成中...",
		"scanning_blockchain":      "ブロックチェーンをスキャン中...",
		"complete":                 "完了",
		"wallet_balance":           "ビットコインウォレット残高",
		"address":                  "アドレス：",
		"total_balance":            "合計残高：",
		"confirmed":                "確認済み：",
		"unconfirmed":              "未確認：",
		"transactions":             "取引：",
		"calculation_details":      "計算詳細：",
		"formula":                  "計算式：",
		"explorer":                 "エクスプローラー：",
		"url":                      "URL：",
		"transaction":              "取引",
		"txid":                     "取引ID：",
		"status":                   "状態：",
		"block":                    "ブロック：",
		"time":                     "時間：",
		"total_input":              "合計入力：",
		"total_output":             "合計出力：",
		"fee":                      "手数料：",
		"size_weight":              "サイズ/重量：",
		"explorer_url":             "エクスプローラーURL：",
		"pending":                  "保留中",
		"confirmed_status":         "確認済み",
		"wallet_has_funds":         "ウォレットに資金があります：%.8f BTC",
		"wallet_zero_balance":      "ウォレットの残高はゼロです",
		"fetching_tx_history":      "アドレスの取引履歴を取得中：%s（制限：%d）",
		"no_transactions":          "アドレス%sの取引が見つかりません",
		"found_transactions":       "アドレス%sで%d件の取引が見つかりました",
		"generating_addresses_msg": "各BIP標準で%d個のアドレスを生成中...",
		"check_balances":           "これらのアドレスの残高を確認しますか？(y/n) >",
		"checking_balances":        "生成されたアドレスの残高を確認中...",
		"checking_address":         "%sアドレスを確認中：%s",
		"found_funds":              "%sアドレスで資金が見つかりました：%s",
		"balance":                  "残高：%.8f BTC",
		"fetching_details":         "詳細情報を取得中...",
		"view_transactions":        "このアドレスの取引を表示しますか？(y/n)：",
		"no_funds_found":           "生成されたアドレスに資金が見つかりませんでした。",
		"random_wallet_scanner":    "ランダムウォレットスキャナー",
		"multi_scanner":            "マルチインスタンスウォレットスキャナー",
		"scanner_warning":          "警告：このツールは教育目的のみです。ランダムウォレットをスキャンして資金を見つける可能性は極めて低いです。違法な目的には使用しないでください。",
		"starting_scanner":         "ランダムウォレットスキャナーを開始しています。結果は%sに保存されます",
		"starting_multi_scanner":   "%d個のスキャナーインスタンスを起動しています。すべてのスキャナーを停止するにはCtrl+Cを押してください。",
		"instance_started":         "インスタンス#%dが起動しました。結果は%sに保存されます",
		"press_ctrl_c":             "スキャナーを停止するにはCtrl+Cを押してください",
		"testing_address":          "テスト中：%s（%s）",
		"wallets_checked":          "確認したウォレット：%d | 見つかったアクティブウォレット：%d",
		"stopping_scanner":         "ウォレットスキャナーを停止中...",
		"stopping_all_scanners":    "すべてのスキャナーインスタンスを停止中...",
		"total_wallets_checked":    "確認したウォレットの総数：%d",
		"total_active_wallets":     "見つかったアクティブウォレットの総数：%d",
		"results_saved":            "結果が保存されました：%s",
		"seed_phrase":              "シードフレーズ：",
		"path":                     "パス：",
		"disconnecting":            "暗号通貨アンダーグラウンドから切断中...",
		"clearing_traces":          "痕跡を消去中...",
		"connection_terminated":    "接続が終了しました。",
		"testing_endpoints":        "Mempool APIエンドポイントの可用性をテスト中...",
		"endpoint":                 "エンドポイント：%s 状態：%s レイテンシー：%s",
		"found_working_endpoints":  "%d個の動作するMempool APIエンドポイントが見つかりました",
		"using_endpoint":           "エンドポイントを使用中：%s",
		"rotating_endpoint":        "API問題のため、%sから%sに切り替えています",
		"endpoint_error":           "エンドポイント%sのエラー：%v",
		"endpoint_status_code":     "エンドポイント%sがステータスコード%dを返しました",
		"json_parse_error":         "%sからJSONを解析できませんでした：%v",
		"generated_addresses":      "生成されたビットコインアドレス",
		"language_selection":       "言語選択",
		"select_language":          "希望する言語を選択してください：",
		"english":                  "英語 (English)",
		"chinese":                  "中国語 (中文)",
		"japanese":                  "日本語",
		"russian":                  "ロシア語 (Русский)",
		"language_selected":        "言語が%sに設定されました",
		"instance_status":          "インスタンス #%d: %s",
		"checking_wallet":          "%sを%s経由で確認中",
		"endpoint_removed":         "継続的な障害のため、エンドポイント%sが削除されました",
		"endpoint_warning":         "警告：エンドポイント%sに問題が発生しています",
		"instance_idle":            "アイドル",
		"instance_active":          "アクティブ",
		"instance_error":           "エラー",
		"instance_success":         "成功",
	},
	LANG_RUSSIAN: {
		"app_title":                "ИНСТРУМЕНТ БИТКОИН-КОШЕЛЬКА v1.0.0",
		"welcome_message":          "ДОБРО ПОЖАЛОВАТЬ В КРИПТОВАЛЮТНОЕ ПОДПОЛЬЕ",
		"risk_warning":             "*** ИСПОЛЬЗУЙТЕ НА СВОЙ СТРАХ И РИСК ***",
		"disconnect":               "ОТКЛЮЧИТЬСЯ",
		"info":                     "ИНФО",
		"success":                  "УСПЕХ",
		"warning":                  "ПРЕДУПРЕЖДЕНИЕ",
		"error":                    "ОШИБКА",
		"scan":                     "СКАНИРОВАНИЕ",
		"stats":                    "СТАТИСТИКА",
		"active_wallet_found":      "НАЙДЕН АКТИВНЫЙ КОШЕЛЕК",
		"menu_check_balance":       "ПРОВЕРИТЬ БАЛАНС КОШЕЛЬКА",
		"menu_view_transactions":   "ПРОСМОТР ИСТОРИИ ТРАНЗАКЦИЙ",
		"menu_generate_addresses":  "ГЕНЕРИРОВАТЬ АДРЕСА ИЗ МНЕМОНИЧЕСКОЙ ФРАЗЫ",
		"menu_scan_wallets":        "СКАНИРОВАТЬ СЛУЧАЙНЫЕ КОШЕЛЬКИ",
		"menu_multi_scan":          "МНОГОПОТОЧНЫЙ СКАНЕР КОШЕЛЬКОВ",
		"menu_exit":                "ВЫХОД",
		"enter_command":            "ВВЕДИТЕ КОМАНДУ >",
		"enter_address":            "Введите биткоин-адрес >",
		"enter_tx_limit":           "Введите количество транзакций для отображения (по умолчанию: 5) >",
		"enter_seed_phrase":        "Введите вашу мнемоническую фразу >",
		"enter_address_count":      "Введите количество адресов для генерации каждого типа (по умолчанию: 5) >",
		"enter_instance_count":     "Введите количество экземпляров сканера для запуска >",
		"no_address":               "Адрес не указан.",
		"no_seed_phrase":           "Мнемоническая фраза не указана.",
		"invalid_seed_phrase":      "Недействительная мнемоническая фраза. Пожалуйста, проверьте и попробуйте снова.",
		"invalid_command":          "Недействительная команда. Пожалуйста, введите число от 1 до 7.",
		"press_enter":              "Нажмите Enter, чтобы вернуться в главное меню...",
		"connecting":               "Подключение к блокчейну...",
		"retrieving_tx_data":       "Получение данных транзакций...",
		"generating_addresses":     "Генерация адресов...",
		"scanning_blockchain":      "Сканирование блокчейна...",
		"complete":                 "ЗАВЕРШЕНО",
		"wallet_balance":           "БАЛАНС БИТКОИН-КОШЕЛЬКА",
		"address":                  "Адрес:",
		"total_balance":            "Общий баланс:",
		"confirmed":                "Подтверждено:",
		"unconfirmed":              "Неподтверждено:",
		"transactions":             "Транзакции:",
		"calculation_details":      "Детали расчета:",
		"formula":                  "Формула:",
		"explorer":                 "Обозреватель:",
		"url":                      "URL:",
		"transaction":              "ТРАНЗАКЦИЯ",
		"txid":                     "TXID:",
		"status":                   "Статус:",
		"block":                    "Блок:",
		"time":                     "Время:",
		"total_input":              "Общий ввод:",
		"total_output":             "Общий вывод:",
		"fee":                      "Комиссия:",
		"size_weight":              "Размер/Вес:",
		"explorer_url":             "URL обозревателя:",
		"pending":                  "В ожидании",
		"confirmed_status":         "Подтверждено",
		"wallet_has_funds":         "Кошелек имеет средства: %.8f BTC",
		"wallet_zero_balance":      "Кошелек имеет нулевой баланс",
		"fetching_tx_history":      "Получение истории транзакций для адреса: %s (лимит: %d)",
		"no_transactions":          "Транзакции не найдены для адреса: %s",
		"found_transactions":       "Найдено %d транзакций для адреса: %s",
		"generating_addresses_msg": "Генерация %d адресов для каждого стандарта BIP...",
		"check_balances":           "Хотите проверить балансы этих адресов? (y/n) >",
		"checking_balances":        "Проверка балансов сгенерированных адресов...",
		"checking_address":         "Проверка %s адреса: %s",
		"found_funds":              "Найдены средства в %s адресе: %s",
		"balance":                  "Баланс: %.8f BTC",
		"fetching_details":         "Получение подробной информации...",
		"view_transactions":        "Хотите просмотреть транзакции для этого адреса? (y/n):",
		"no_funds_found":           "Средства не найдены ни в одном из сгенерированных адресов.",
		"random_wallet_scanner":    "СКАНЕР СЛУЧАЙНЫХ КОШЕЛЬКОВ",
		"multi_scanner":            "МНОГОПОТОЧНЫЙ СКАНЕР КОШЕЛЬКОВ",
		"scanner_warning":          "ПРЕДУПРЕЖДЕНИЕ: Этот инструмент предназначен только для образовательных целей. Сканирование случайных кошельков крайне маловероятно найдет какие-либо средства. Не используйте для незаконных целей.",
		"starting_scanner":         "Запуск сканера случайных кошельков. Результаты будут сохранены в %s",
		"starting_multi_scanner":   "Запуск %d экземпляров сканера. Нажмите Ctrl+C, чтобы остановить все сканеры.",
		"instance_started":         "Экземпляр #%d запущен. Результаты будут сохранены в %s",
		"press_ctrl_c":             "Нажмите Ctrl+C, чтобы остановить сканер",
		"testing_address":          "Тестирование: %s (%s)",
		"wallets_checked":          "Проверено кошельков: %d | Найдено активных кошельков: %d",
		"stopping_scanner":         "Остановка сканера кошельков...",
		"stopping_all_scanners":    "Остановка всех экземпляров сканера...",
		"total_wallets_checked":    "Всего проверено кошельков: %d",
		"total_active_wallets":     "Всего найдено активных кошельков: %d",
		"results_saved":            "Результаты сохранены в: %s",
		"seed_phrase":              "Мнемоническая фраза:",
		"path":                     "Путь:",
		"disconnecting":            "Отключение от криптовалютного подполья...",
		"clearing_traces":          "Очистка следов...",
		"connection_terminated":    "Соединение прервано.",
		"testing_endpoints":        "Тестирование доступности конечных точек Mempool API...",
		"endpoint":                 "Конечная точка: %s Статус: %s Задержка: %s",
		"found_working_endpoints":  "Найдено %d работающих конечных точек Mempool API",
		"using_endpoint":           "Использование конечной точки: %s",
		"rotating_endpoint":        "Переключение с %s на %s из-за проблем с API",
		"endpoint_error":           "Ошибка с конечной точкой %s: %v",
		"endpoint_status_code":     "Конечная точка %s вернула код состояния %d",
		"json_parse_error":         "Не удалось разобрать JSON из %s: %v",
		"generated_addresses":      "СГЕНЕРИРОВАННЫЕ БИТКОИН-АДРЕСА",
		"language_selection":       "ВЫБОР ЯЗЫКА",
		"select_language":          "Выберите предпочитаемый язык:",
		"english":                  "Английский (English)",
		"chinese":                  "Китайский (中文)",
		"japanese":                  "Японский (日本語)",
		"russian":                  "Русский",
		"language_selected":        "Язык установлен на %s",
		"instance_status":          "Экземпляр #%d: %s",
		"checking_wallet":          "Проверка %s через %s",
		"endpoint_removed":         "Конечная точка %s удалена из-за постоянных сбоев",
		"endpoint_warning":         "Предупреждение: Конечная точка %s испытывает проблемы",
		"instance_idle":            "Ожидание",
		"instance_active":          "Активен",
		"instance_error":           "Ошибка",
		"instance_success":         "Успех",
	},
}

// Animation patterns for enhanced UI
var (
	loadingPatterns = []string{
		"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
	}
	glitchChars = []string{
		"█", "▓", "▒", "░", "▄", "▀", "■", "□", "▪", "▫", "◢", "◣", "◤", "◥",
		" ", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▉", "▊", "▋", "▌", "▍", "▎", "▏",
	}
	matrixChars = []string{
		"ｱ", "ｲ", "ｳ", "ｴ", "ｵ", "ｶ", "ｷ", "ｸ", "ｹ", "ｺ", "ｻ", "ｼ", "ｽ", "ｾ", "ｿ",
		"ﾀ", "ﾁ", "ﾂ", "ﾃ", "ﾄ", "ﾅ", "ﾆ", "ﾇ", "ﾈ", "ﾉ", "ﾊ", "ﾋ", "ﾌ", "ﾍ", "ﾎ",
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F",
	}
)

// WalletData represents the response from the Mempool.space API
type WalletData struct {
	Address     string `json:"address"`
	ChainStats  Stats  `json:"chain_stats"`
	MempoolStats Stats  `json:"mempool_stats"`
}

// Stats represents the statistics for a wallet
type Stats struct {
	FundedTxoSum int64 `json:"funded_txo_sum"`
	SpentTxoSum  int64 `json:"spent_txo_sum"`
	TxCount      int   `json:"tx_count"`
}

// Transaction represents a Bitcoin transaction
type Transaction struct {
	Txid   string        `json:"txid"`
	Status TransStatus   `json:"status"`
	Fee    int64         `json:"fee"`
	Size   int           `json:"size"`
	Weight int           `json:"weight"`
	Vin    []TxInput     `json:"vin"`
	Vout   []TxOutput    `json:"vout"`
}

// TransStatus represents the status of a transaction
type TransStatus struct {
	Confirmed   bool  `json:"confirmed"`
	BlockHeight int   `json:"block_height"`
	BlockTime   int64 `json:"block_time"`
}

// TxInput represents an input in a transaction
type TxInput struct {
	Prevout TxPrevout `json:"prevout"`
}

// TxPrevout represents the previous output referenced by an input
type TxPrevout struct {
	Value int64 `json:"value"`
}

// TxOutput represents an output in a transaction
type TxOutput struct {
	Value int64 `json:"value"`
}

// DetailedStats represents detailed wallet statistics
type DetailedStats struct {
	ConfirmedFunded    int64
	ConfirmedSpent     int64
	ConfirmedBalance   int64
	UnconfirmedFunded  int64
	UnconfirmedSpent   int64
	UnconfirmedBalance int64
	TotalBalanceSats   int64
	TxCount            int
}

// ScannerStats tracks statistics for a wallet scanner instance
type ScannerStats struct {
	WalletsChecked     int
	ActiveWalletsFound int
	mutex              sync.Mutex
}

// InstanceStatus tracks the status of a scanner instance
type InstanceStatus struct {
	ID              int
	Status          string // "idle", "active", "error", "success"
	CurrentAddress  string
	CurrentBIPType  string
	CurrentEndpoint string
	LastError       string
	LastErrorTime   time.Time
	mutex           sync.Mutex
}

// Update the instance status
func (s *InstanceStatus) Update(status, address, bipType, endpoint string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Status = status
	s.CurrentAddress = address
	s.CurrentBIPType = bipType
	s.CurrentEndpoint = endpoint
}

// Set an error for the instance
func (s *InstanceStatus) SetError(err string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// Only update if it's a new error or if the previous error was more than 5 minutes ago
	if s.LastError != err || time.Since(s.LastErrorTime) > 5*time.Minute {
		s.Status = "error"
		s.LastError = err
		s.LastErrorTime = time.Now()
	}
}

// Get the current status
func (s *InstanceStatus) GetStatus() (string, string, string, string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.Status, s.CurrentAddress, s.CurrentBIPType, s.CurrentEndpoint
}

// Increment the wallets checked counter safely
func (s *ScannerStats) IncrementWalletsChecked() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.WalletsChecked++
}

// Increment the active wallets found counter safely
func (s *ScannerStats) IncrementActiveWalletsFound() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.ActiveWalletsFound++
}

// Get the current statistics safely
func (s *ScannerStats) GetStats() (int, int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.WalletsChecked, s.ActiveWalletsFound
}

// Translation function to get text in the current language
func T(key string) string {
	if text, ok := translations[currentLanguage][key]; ok {
		return text
	}
	// Fallback to English if translation not found
	if text, ok := translations[LANG_ENGLISH][key]; ok {
		return text
	}
	return key
}

// Tf is like T but with formatting
func Tf(key string, args ...interface{}) string {
	return fmt.Sprintf(T(key), args...)
}

// Print functions for formatted output
func printInfo(message string) {
	fmt.Printf("%s %s\n", green("["+T("info")+"]"), message)
}

func printSuccess(message string) {
	fmt.Printf("%s %s\n", greenBold("["+T("success")+"]"), message)
}

func printWarning(message string) {
	fmt.Printf("%s %s\n", amber("["+T("warning")+"]"), message)
}

func printError(message string) {
	fmt.Printf("%s %s\n", red("["+T("error")+"]"), message)
}

// typeText simulates typing text for retro effect
func typeText(text string, delay time.Duration) {
	for _, char := range text {
		fmt.Print(string(char))
		time.Sleep(delay)
	}
	fmt.Println()
}

// clearScreen clears the terminal
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// clearLine clears the current line
func clearLine() {
	fmt.Print("\r\033[K")
}

// moveCursorUp moves the cursor up n lines
func moveCursorUp(n int) {
	fmt.Printf("\033[%dA", n)
}

// moveCursorDown moves the cursor down n lines
func moveCursorDown(n int) {
	fmt.Printf("\033[%dB", n)
}

// getTerminalWidth gets the width of the terminal
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // Default width
	}
	return width
}

// showMatrixEffect shows a matrix-like animation effect
func showMatrixEffect(duration time.Duration) {
	start := time.Now()
	width, height := 80, 20
	
	// Create a matrix of characters
	matrix := make([][]string, height)
	for i := range matrix {
		matrix[i] = make([]string, width)
		for j := range matrix[i] {
			matrix[i][j] = " "
		}
	}
	
	// Create a slice to track the position of each falling character
	positions := make([]int, width)
	for i := range positions {
		positions[i] = rand.Intn(height * 2) - height
	}
	
	// Run the animation until the duration is reached
	for time.Since(start) < duration {
		// Clear the screen
		fmt.Print("\033[H")
		
		// Update the matrix
		for i := 0; i < width; i++ {
			// Move the character down
			positions[i]++
			if positions[i] >= height {
				positions[i] = -rand.Intn(height)
			}
			
			// Draw the character if it's in the visible area
			if positions[i] >= 0 && positions[i] < height {
				matrix[positions[i]][i] = matrixChars[rand.Intn(len(matrixChars))]
				
				// Add a trail
				for j := 1; j <= 5; j++ {
					if positions[i]-j >= 0 && positions[i]-j < height {
						if j == 1 {
							matrix[positions[i]-j][i] = green(matrixChars[rand.Intn(len(matrixChars))])
						} else {
							opacity := 5 - j + 1
							if rand.Intn(10) < opacity {
								matrix[positions[i]-j][i] = cyan(matrixChars[rand.Intn(len(matrixChars))])
							} else {
								matrix[positions[i]-j][i] = " "
							}
						}
					}
				}
			}
		}
		
		// Draw the matrix
		for i := 0; i < height; i++ {
			for j := 0; j < width; j++ {
				if i == positions[j] {
					fmt.Print(green(matrix[i][j]))
				} else {
					fmt.Print(matrix[i][j])
				}
			}
			fmt.Println()
		}
		
		time.Sleep(50 * time.Millisecond)
	}
	
	clearScreen()
}

// showGlitch shows a random glitch effect
func showGlitch() {
	// Determine the number of glitch lines
	lines := rand.Intn(5) + 3
	
	for i := 0; i < lines; i++ {
		// Determine the length of this glitch line
		glitchLength := rand.Intn(40) + 20
		
		// Choose a random color for this line
		colorFunc := green
		switch rand.Intn(5) {
		case 0:
			colorFunc = amber
		case 1:
			colorFunc = cyan
		case 2:
			colorFunc = red
		case 3:
			colorFunc = magenta
		}
		
		// Generate the glitch line
		for j := 0; j < glitchLength; j++ {
			fmt.Print(colorFunc(glitchChars[rand.Intn(len(glitchChars))]))
		}
		fmt.Println()
		
		// Add a small delay between lines
		time.Sleep(50 * time.Millisecond)
	}
	
	// Add a final delay
	time.Sleep(100 * time.Millisecond)
}

// showLoadingBar shows a retro loading bar with animation
func showLoadingBar(message string, duration time.Duration) {
	fmt.Print(message)
	steps := 20
	stepDuration := duration / time.Duration(steps)
	
	for i := 0; i < steps; i++ {
		// Choose a random color occasionally for a glitch effect
		colorFunc := green
		if rand.Intn(10) == 0 {
			switch rand.Intn(3) {
			case 0:
				colorFunc = amber
			case 1:
				colorFunc = cyan
			case 2:
				colorFunc = magenta
			}
		}
		
		fmt.Print(colorFunc("█"))
		
		// Occasionally show a loading spinner
		if i < steps-1 && rand.Intn(5) == 0 {
			for j := 0; j < 3; j++ {
				time.Sleep(50 * time.Millisecond)
				fmt.Print("\b" + colorFunc(loadingPatterns[rand.Intn(len(loadingPatterns))]))
			}
			fmt.Print("\b")
		}
		
		time.Sleep(stepDuration)
	}
	fmt.Println(" " + T("complete"))
}

// showSpinner shows an animated spinner with the given message
func showSpinner(message string, duration time.Duration) {
	start := time.Now()
	i := 0
	
	for time.Since(start) < duration {
		pattern := loadingPatterns[i%len(loadingPatterns)]
		fmt.Printf("\r%s %s", cyan(pattern), message)
		time.Sleep(100 * time.Millisecond)
		i++
	}
	fmt.Println()
}

// incrementEndpointFailure increments the failure count for an endpoint
func incrementEndpointFailure(endpoint string) {
	endpointFailureMutex.Lock()
	defer endpointFailureMutex.Unlock()
	
	endpointFailures[endpoint]++
	
	// If the endpoint has failed too many times, remove it
	if endpointFailures[endpoint] >= 5 {
		removeEndpoint(endpoint)
	}
}

// removeEndpoint removes an endpoint from the list
func removeEndpoint(endpoint string) {
	endpointMutex.Lock()
	defer endpointMutex.Unlock()
	
	// Find the endpoint in the list
	for i, ep := range mempoolEndpoints {
		if ep == endpoint {
			// Remove the endpoint
			mempoolEndpoints = append(mempoolEndpoints[:i], mempoolEndpoints[i+1:]...)
			
			// If we removed the current endpoint, update the index
			if i <= currentEndpointIndex {
				currentEndpointIndex = currentEndpointIndex % len(mempoolEndpoints)
				currentEndpoint = mempoolEndpoints[currentEndpointIndex]
			}
			
			printWarning(Tf("endpoint_removed", endpoint))
			break
		}
	}
}

// getNextEndpoint rotates to the next Mempool API endpoint
func getNextEndpoint() string {
	endpointMutex.Lock()
	defer endpointMutex.Unlock()
	
	if len(mempoolEndpoints) == 0 {
		return ""
	}
	
	currentEndpointIndex = (currentEndpointIndex + 1) % len(mempoolEndpoints)
	currentEndpoint = mempoolEndpoints[currentEndpointIndex]
	return currentEndpoint
}

// getCurrentEndpoint gets the current Mempool API endpoint
func getCurrentEndpoint() string {
	endpointMutex.Lock()
	defer endpointMutex.Unlock()
	
	if currentEndpoint == "" && len(mempoolEndpoints) > 0 {
		currentEndpoint = mempoolEndpoints[currentEndpointIndex]
	}
	return currentEndpoint
}

// rotateEndpointIfNeeded checks if we need to rotate to a new endpoint
func rotateEndpointIfNeeded(statusCode int, err error) bool {
	// Rotate if we got rate limited or other specific errors
	if statusCode == 429 || statusCode >= 500 || 
	   (err != nil && (strings.Contains(err.Error(), "timeout") || 
					  strings.Contains(err.Error(), "connection refused"))) {
		oldEndpoint := getCurrentEndpoint()
		incrementEndpointFailure(oldEndpoint)
		newEndpoint := getNextEndpoint()
		
		if oldEndpoint != newEndpoint {
			printWarning(Tf("rotating_endpoint", oldEndpoint, newEndpoint))
			return true
		}
	}
	return false
}

// testAndSortEndpoints tests all endpoints and sorts them by availability and latency
func testAndSortEndpoints() {
	printInfo(T("testing_endpoints"))
	
	type endpointStatus struct {
		endpoint string
		latency  time.Duration
		working  bool
	}
	
	results := make([]endpointStatus, len(mempoolEndpoints))
	var wg sync.WaitGroup
	
	for i, endpoint := range mempoolEndpoints {
		wg.Add(1)
		go func(i int, endpoint string) {
			defer wg.Done()
			
			start := time.Now()
			url := fmt.Sprintf("https://%s/api/blocks/tip/height", endpoint)
			
			client := &http.Client{
				Timeout: 5 * time.Second,
			}
			resp, err := client.Get(url)
			
			if err != nil {
				results[i] = endpointStatus{endpoint, 0, false}
				return
			}
			defer resp.Body.Close()
			
			latency := time.Since(start)
			results[i] = endpointStatus{endpoint, latency, resp.StatusCode == http.StatusOK}
		}(i, endpoint)
	}
	
	wg.Wait()
	
	// Sort endpoints: working ones first, then by latency
	sort.Slice(results, func(i, j int) bool {
		if results[i].working != results[j].working {
			return results[i].working
		}
		return results[i].latency < results[j].latency
	})
	
	// Update the endpoints list with sorted results
	newEndpoints := make([]string, 0, len(results))
	for _, result := range results {
		status := "✓"
		if !result.working {
			status = "✗"
		}
		latency := "N/A"
		if result.working {
			latency = fmt.Sprintf("%dms", result.latency.Milliseconds())
		}
		printInfo(Tf("endpoint", result.endpoint, status, latency))
		
		if result.working {
			newEndpoints = append(newEndpoints, result.endpoint)
		}
	}
	
	// Add non-working endpoints at the end
	for _, result := range results {
		if !result.working {
			newEndpoints = append(newEndpoints, result.endpoint)
		}
	}
	
	// Update the global endpoints list
	endpointMutex.Lock()
	mempoolEndpoints = newEndpoints
	currentEndpointIndex = 0
	if len(newEndpoints) > 0 {
		currentEndpoint = newEndpoints[0]
	}
	endpointMutex.Unlock()
	
	// Reset endpoint failures
	endpointFailureMutex.Lock()
	endpointFailures = make(map[string]int)
	endpointFailureMutex.Unlock()
	
	printSuccess(Tf("found_working_endpoints", len(newEndpoints)))
	if len(newEndpoints) > 0 {
		printInfo(Tf("using_endpoint", currentEndpoint))
	}
}

// fetchWalletData fetches wallet data from Mempool API with endpoint rotation
func fetchWalletData(address string) (*WalletData, error) {
	endpoint := getCurrentEndpoint()
	if endpoint == "" {
		return nil, fmt.Errorf("no available endpoints")
	}
	
	url := fmt.Sprintf("https://%s/api/address/%s", endpoint, address)
	
	// Try using http.Get
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
	
	// Check if we need to rotate endpoints
	if err != nil {
		incrementEndpointFailure(endpoint)
		rotateEndpointIfNeeded(0, err)
		return nil, fmt.Errorf("failed to fetch wallet data: %v", err)
	}
	
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		incrementEndpointFailure(endpoint)
		rotateEndpointIfNeeded(resp.StatusCode, nil)
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}
	
	output, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	
	var data WalletData
	if err := json.Unmarshal(output, &data); err != nil {
		// If we can't parse the JSON, it might be an error response or invalid data
		incrementEndpointFailure(endpoint)
		rotateEndpointIfNeeded(0, fmt.Errorf("invalid JSON"))
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}
	
	return &data, nil
}

// fetchWalletTransactions fetches transaction history for a wallet address
func fetchWalletTransactions(address string, limit int) ([]Transaction, error) {
	endpoint := getCurrentEndpoint()
	if endpoint == "" {
		return nil, fmt.Errorf("no available endpoints")
	}
	
	url := fmt.Sprintf("https://%s/api/address/%s/txs", endpoint, address)
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
	
	if err != nil {
		incrementEndpointFailure(endpoint)
		rotateEndpointIfNeeded(0, err)
		return nil, fmt.Errorf("failed to fetch transaction history: %v", err)
	}
	
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		incrementEndpointFailure(endpoint)
		rotateEndpointIfNeeded(resp.StatusCode, nil)
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}
	
	output, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	
	var transactions []Transaction
	if err := json.Unmarshal(output, &transactions); err != nil {
		incrementEndpointFailure(endpoint)
		rotateEndpointIfNeeded(0, fmt.Errorf("invalid JSON"))
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}
	
	// Limit the number of transactions
	if len(transactions) > limit {
		transactions = transactions[:limit]
	}
	
	return transactions, nil
}

// calculateBalance calculates the total balance from wallet data
func calculateBalance(data *WalletData) (float64, DetailedStats) {
	// Extract values from the data
	fChain := data.ChainStats.FundedTxoSum
	sChain := data.ChainStats.SpentTxoSum
	
	fMempool := data.MempoolStats.FundedTxoSum
	sMempool := data.MempoolStats.SpentTxoSum
	
	// Calculate confirmed and unconfirmed balances
	confirmedBalance := fChain - sChain
	unconfirmedBalance := fMempool - sMempool
	
	// Calculate total balance in satoshis
	totalBalanceSats := confirmedBalance + unconfirmedBalance
	
	// Convert to BTC
	totalBalanceBtc := float64(totalBalanceSats) / 100000000.0
	
	// Prepare detailed stats
	stats := DetailedStats{
		ConfirmedFunded:    fChain,
		ConfirmedSpent:     sChain,
		ConfirmedBalance:   confirmedBalance,
		UnconfirmedFunded:  fMempool,
		UnconfirmedSpent:   sMempool,
		UnconfirmedBalance: unconfirmedBalance,
		TotalBalanceSats:   totalBalanceSats,
		TxCount:            data.ChainStats.TxCount + data.MempoolStats.TxCount,
	}
	
	return totalBalanceBtc, stats
}

// safeRepeat safely repeats a string n times, ensuring n is not negative
func safeRepeat(s string, n int) string {
	if n < 0 {
		n = 0
	}
	return strings.Repeat(s, n)
}

// formatWithCommas formats a number with commas for thousands
func formatWithCommas(n int64) string {
	in := strconv.FormatInt(n, 10)
	out := make([]byte, 0, len(in)+(len(in)-1)/3)
	
	// Process the number from right to left
	for i, c := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	
	return string(out)
}

// getVisualLength returns the visual length of a string, accounting for multi-byte characters
func getVisualLength(s string) int {
	return utf8.RuneCountInString(s)
}

// centerText centers text in a field of given width
func centerText(text string, width int) string {
	textLen := getVisualLength(text)
	if textLen >= width {
		return text
	}
	
	leftPadding := (width - textLen) / 2
	rightPadding := width - textLen - leftPadding
	
	return safeRepeat(" ", leftPadding) + text + safeRepeat(" ", rightPadding)
}

// createBox creates a box with the given width, title, and content
func createBox(width int, title string, content []string) string {
	var output strings.Builder
	
	// Top border
	output.WriteString(fmt.Sprintf("\n%s\n", green("╔"+safeRepeat("═", width-2)+"╗")))
	
	// Title
	if title != "" {
		centeredTitle := centerText(whiteBold(title), width-4)
		output.WriteString(fmt.Sprintf("%s %s %s\n", green("║"), centeredTitle, green("║")))
		output.WriteString(fmt.Sprintf("%s\n", green("╠"+safeRepeat("═", width-2)+"╣")))
	}
	
	// Content
	for _, line := range content {
		// Calculate padding to ensure the line is exactly the right width
		lineLen := getVisualLength(line)
		padding := width - 4 - lineLen
		if padding < 0 {
			padding = 0
		}
		
		output.WriteString(fmt.Sprintf("%s %s%s %s\n", 
			green("║"), 
			line, 
			safeRepeat(" ", padding),
			green("║")))
	}
	
	// Bottom border
	output.WriteString(fmt.Sprintf("%s", green("╚"+safeRepeat("═", width-2)+"╝")))
	
	return output.String()
}

// formatBalanceOutput formats the balance output for display
func formatBalanceOutput(address string, balance float64, stats DetailedStats) string {
	// Determine color based on balance
	balanceColor := amber
	if balance > 0 {
		balanceColor = green
	}
	
	// Truncate address if too long
	displayAddr := address
	if getVisualLength(address) > 24 {
		displayAddr = address[:16]+"..."+address[len(address)-4:]
	}
	
	// Prepare content lines
	var content []string
	
	// Address line
	content = append(content, fmt.Sprintf("%s %s", amber(T("address")), displayAddr))
	
	// Divider
	content = append(content, green(safeRepeat("─", DISPLAY_WIDTH-4)))
	
	// Balance lines
	balanceStr := fmt.Sprintf("%.8f BTC", balance)
	content = append(content, fmt.Sprintf("%s %s", whiteBold(balanceColor(T("total_balance"))), balanceStr))
	
	confirmedBtc := float64(stats.ConfirmedBalance) / 100000000.0
	confirmedStr := fmt.Sprintf("%.8f BTC", confirmedBtc)
	content = append(content, fmt.Sprintf("%s %s", amber(T("confirmed")), confirmedStr))
	
	if stats.UnconfirmedBalance != 0 {
		unconfirmedBtc := float64(stats.UnconfirmedBalance) / 100000000.0
		unconfColor := green
		if unconfirmedBtc < 0 {
			unconfColor = red
		}
		unconfirmedStr := fmt.Sprintf("%.8f BTC", unconfirmedBtc)
		content = append(content, fmt.Sprintf("%s %s", unconfColor(T("unconfirmed")), unconfirmedStr))
	}
	
	txCountStr := fmt.Sprintf("%d", stats.TxCount)
	content = append(content, fmt.Sprintf("%s %s", amber(T("transactions")), txCountStr))
	
	// Divider
	content = append(content, green(safeRepeat("─", DISPLAY_WIDTH-4)))
	
	// Calculation details
	content = append(content, cyan(T("calculation_details")))
	
	fChainFormatted := formatWithCommas(stats.ConfirmedFunded)
	content = append(content, fmt.Sprintf("%s %s", amber("F_chain:"), fChainFormatted))
	
	sChainFormatted := formatWithCommas(stats.ConfirmedSpent)
	content = append(content, fmt.Sprintf("%s %s", amber("S_chain:"), sChainFormatted))
	
	if stats.UnconfirmedFunded > 0 || stats.UnconfirmedSpent > 0 {
		fMempoolFormatted := formatWithCommas(stats.UnconfirmedFunded)
		content = append(content, fmt.Sprintf("%s %s", amber("F_mempool:"), fMempoolFormatted))
		
		sMempoolFormatted := formatWithCommas(stats.UnconfirmedSpent)
		content = append(content, fmt.Sprintf("%s %s", amber("S_mempool:"), sMempoolFormatted))
	}
	
	content = append(content, fmt.Sprintf("%s ((F_chain - S_chain) + (F_mempool - S_mempool)) / 100M", cyan(T("formula"))))
	
	// Divider
	content = append(content, green(safeRepeat("─", DISPLAY_WIDTH-4)))
	
	// Explorer info
	content = append(content, fmt.Sprintf("%s Mempool.space", cyan(T("explorer"))))
	
	explorerURL := fmt.Sprintf("https://mempool.space/address/%s", address)
	displayURL := explorerURL
	if getVisualLength(explorerURL) > 38 {
		displayURL = explorerURL[:38]
	}
	content = append(content, fmt.Sprintf("%s %s", cyan(T("url")), displayURL))
	
	return createBox(DISPLAY_WIDTH, T("wallet_balance"), content)
}

// formatTransactionOutput formats a transaction for display
func formatTransactionOutput(tx Transaction, index int) string {
	// Extract transaction details
	txid := tx.Txid
	confirmed := tx.Status.Confirmed
	blockHeight := tx.Status.BlockHeight
	blockTime := tx.Status.BlockTime
	
	// Calculate fee in BTC
	feeSats := tx.Fee
	feeBtc := float64(feeSats) / 100000000.0
	
	// Format timestamp
	var timestamp string
	if blockTime > 0 {
		t := time.Unix(blockTime, 0)
		timestamp = t.Format("2006-01-02 15:04:05")
	} else {
		timestamp = T("pending")
	}
	
	// Calculate total input and output values
	var totalInput int64
	for _, input := range tx.Vin {
		totalInput += input.Prevout.Value
	}
	
	var totalOutput int64
	for _, output := range tx.Vout {
		totalOutput += output.Value
	}
	
	// Convert to BTC
	totalInputBtc := float64(totalInput) / 100000000.0
	totalOutputBtc := float64(totalOutput) / 100000000.0
	
	// Determine status color and text
	statusColor := amber
	statusText := T("pending")
	if confirmed {
		statusColor = green
		statusText = T("confirmed_status")
	}
	
	// Prepare content lines
	var content []string
	
	// Transaction ID
	displayTxid := txid
	if getVisualLength(txid) > 35 {
		displayTxid = txid[:32] + "..."
	}
	content = append(content, fmt.Sprintf("%s %s", amber(T("txid")), displayTxid))
	
	// Status
	content = append(content, fmt.Sprintf("%s %s", amber(T("status")), statusColor(statusText)))
	
	// Block height if confirmed
	if confirmed {
		blockHeightStr := fmt.Sprintf("%d", blockHeight)
		content = append(content, fmt.Sprintf("%s %s", amber(T("block")), blockHeightStr))
	}
	
	// Timestamp
	content = append(content, fmt.Sprintf("%s %s", amber(T("time")), timestamp))
	
	// Divider
	content = append(content, green(safeRepeat("─", DISPLAY_WIDTH-4)))
	
	// Amounts
	totalInputStr := fmt.Sprintf("%.8f BTC", totalInputBtc)
	content = append(content, fmt.Sprintf("%s %s", green(T("total_input")), totalInputStr))
	
	totalOutputStr := fmt.Sprintf("%.8f BTC", totalOutputBtc)
	content = append(content, fmt.Sprintf("%s %s", red(T("total_output")), totalOutputStr))
	
	feeStr := fmt.Sprintf("%.8f BTC", feeBtc)
	content = append(content, fmt.Sprintf("%s %s", amber(T("fee")), feeStr))
	
	// Size information
	sizeWeightStr := fmt.Sprintf("%d bytes / %d WU", tx.Size, tx.Weight)
	content = append(content, fmt.Sprintf("%s %s", amber(T("size_weight")), sizeWeightStr))
	
	// Divider
	content = append(content, green(safeRepeat("─", DISPLAY_WIDTH-4)))
	
	// Explorer link
	explorerURL := fmt.Sprintf("https://mempool.space/tx/%s", txid)
	displayURL := explorerURL
	if getVisualLength(explorerURL) > 35 {
		displayURL = explorerURL[:35]
	}
	content = append(content, fmt.Sprintf("%s %s", cyan(T("explorer_url")), displayURL))
	
	return createBox(DISPLAY_WIDTH, fmt.Sprintf("%s #%d", T("transaction"), index+1), content)
}

// checkWalletBalance checks the balance of a Bitcoin wallet and displays the results
func checkWalletBalance(address string) {
	showLoadingBar(T("connecting"), 2*time.Second)
	
	// Fetch wallet data
	data, err := fetchWalletData(address)
	if err != nil {
		printError(fmt.Sprintf("Failed to check wallet balance: %v", err))
		return
	}
	
	// Calculate balance
	balance, stats := calculateBalance(data)
	
	// Show a glitch for effect
	showGlitch()
	
	// Format and display the output
	output := formatBalanceOutput(address, balance, stats)
	fmt.Println(output)
	
	// Print summary message
	if balance > 0 {
		printSuccess(Tf("wallet_has_funds", balance))
	} else {
		printWarning(T("wallet_zero_balance"))
	}
}

// displayTransactions fetches and displays transaction history for a wallet address
func displayTransactions(address string, limit int) {
	printInfo(Tf("fetching_tx_history", address, limit))
	showLoadingBar(T("retrieving_tx_data"), 3*time.Second)
	
	// Fetch transaction data
	transactions, err := fetchWalletTransactions(address, limit)
	if err != nil {
		printError(fmt.Sprintf("Failed to fetch transaction history: %v", err))
		return
	}
	
	if len(transactions) == 0 {
		printWarning(Tf("no_transactions", address))
		return
	}
	
	printSuccess(Tf("found_transactions", len(transactions), address))
	
	// Display each transaction
	for i, tx := range transactions {
		// Show a glitch for effect
		if i > 0 {
			showGlitch()
		}
		
		output := formatTransactionOutput(tx, i)
		fmt.Println(output)
		
		// Add a small delay between transactions for readability
		if i < len(transactions)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// validateSeedPhrase validates a BIP39 seed phrase
func validateSeedPhrase(seedPhrase string) bool {
	return bip39.IsMnemonicValid(seedPhrase)
}

// seedToRootKey converts a seed phrase to a BIP32 root key
func seedToRootKey(seedPhrase string) []byte {
	seed := bip39.NewSeed(seedPhrase, "")
	return seed
}

// deriveBip32Address derives a BIP32 address from a seed
func deriveBip32Address(seed []byte, index int) (string, error) {
	// Create master key from seed
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to create master key: %v", err)
	}
	
	// Derive path m/0'/0/index
	purpose, err := masterKey.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return "", fmt.Errorf("failed to derive purpose: %v", err)
	}
	
	account, err := purpose.Derive(0)
	if err != nil {
		return "", fmt.Errorf("failed to derive account: %v", err)
	}
	
	child, err := account.Derive(uint32(index))
	if err != nil {
		return "", fmt.Errorf("failed to derive child: %v", err)
	}
	
	// Get the public key
	pubKey, err := child.ECPubKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %v", err)
	}
	
	// Create P2PKH address
	address, err := btcutil.NewAddressPubKey(pubKey.SerializeCompressed(), &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to create address: %v", err)
	}
	
	return address.EncodeAddress(), nil
}

// deriveBip44Address derives a BIP44 address from a seed
func deriveBip44Address(seed []byte, account, change, index int) (string, error) {
	// Create master key from seed
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to create master key: %v", err)
	}
	
	// BIP44 path: m/44'/0'/account'/change/index
	purpose, err := masterKey.Derive(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return "", fmt.Errorf("failed to derive purpose: %v", err)
	}
	
	coinType, err := purpose.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return "", fmt.Errorf("failed to derive coin type: %v", err)
	}
	
	accountKey, err := coinType.Derive(hdkeychain.HardenedKeyStart + uint32(account))
	if err != nil {
		return "", fmt.Errorf("failed to derive account: %v", err)
	}
	
	changeKey, err := accountKey.Derive(uint32(change))
	if err != nil {
		return "", fmt.Errorf("failed to derive change: %v", err)
	}
	
	indexKey, err := changeKey.Derive(uint32(index))
	if err != nil {
		return "", fmt.Errorf("failed to derive index: %v", err)
	}
	
	// Get the public key
	pubKey, err := indexKey.ECPubKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %v", err)
	}
	
	// Create P2PKH address
	address, err := btcutil.NewAddressPubKey(pubKey.SerializeCompressed(), &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to create address: %v", err)
	}
	
	return address.EncodeAddress(), nil
}

// hash160 performs RIPEMD160(SHA256(data))
func hash160(data []byte) []byte {
	h := sha256.Sum256(data)
	ripemd := ripemd160.New()
	ripemd.Write(h[:])
	return ripemd.Sum(nil)
}

// deriveBip49Address derives a BIP49 (SegWit-compatible) address from a seed
func deriveBip49Address(seed []byte, account, change, index int) (string, error) {
	// Create master key from seed
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to create master key: %v", err)
	}
	
	// BIP49 path: m/49'/0'/account'/change/index
	purpose, err := masterKey.Derive(hdkeychain.HardenedKeyStart + 49)
	if err != nil {
		return "", fmt.Errorf("failed to derive purpose: %v", err)
	}
	
	coinType, err := purpose.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return "", fmt.Errorf("failed to derive coin type: %v", err)
	}
	
	accountKey, err := coinType.Derive(hdkeychain.HardenedKeyStart + uint32(account))
	if err != nil {
		return "", fmt.Errorf("failed to derive account: %v", err)
	}
	
	changeKey, err := accountKey.Derive(uint32(change))
	if err != nil {
		return "", fmt.Errorf("failed to derive change: %v", err)
	}
	
	indexKey, err := changeKey.Derive(uint32(index))
	if err != nil {
		return "", fmt.Errorf("failed to derive index: %v", err)
	}
	
	// Get the public key
	pubKey, err := indexKey.ECPubKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %v", err)
	}
	
	// Create P2SH-wrapped P2WPKH address
	pubKeyHash := hash160(pubKey.SerializeCompressed())
	
	// Create the witness program
	witnessProgram := []byte{0x00, 0x14}
	witnessProgram = append(witnessProgram, pubKeyHash...)
	
	// Hash the witness program
	scriptHash := hash160(witnessProgram)
	
	// Create P2SH address
	address, err := btcutil.NewAddressScriptHash(scriptHash, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to create address: %v", err)
	}
	
	return address.EncodeAddress(), nil
}

// deriveBip84Address derives a BIP84 (Native SegWit) address from a seed
func deriveBip84Address(seed []byte, account, change, index int) (string, error) {
	// Create master key from seed
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to create master key: %v", err)
	}
	
	// BIP84 path: m/84'/0'/account'/change/index
	purpose, err := masterKey.Derive(hdkeychain.HardenedKeyStart + 84)
	if err != nil {
		return "", fmt.Errorf("failed to derive purpose: %v", err)
	}
	
	coinType, err := purpose.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return "", fmt.Errorf("failed to derive coin type: %v", err)
	}
	
	accountKey, err := coinType.Derive(hdkeychain.HardenedKeyStart + uint32(account))
	if err != nil {
		return "", fmt.Errorf("failed to derive account: %v", err)
	}
	
	changeKey, err := accountKey.Derive(uint32(change))
	if err != nil {
		return "", fmt.Errorf("failed to derive change: %v", err)
	}
	
	indexKey, err := changeKey.Derive(uint32(index))
	if err != nil {
		return "", fmt.Errorf("failed to derive index: %v", err)
	}
	
	// Get the public key
	pubKey, err := indexKey.ECPubKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %v", err)
	}
	
	// Create native SegWit (Bech32) address
	pubKeyHash := hash160(pubKey.SerializeCompressed())
	address, err := btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
	if err != nil {
		return "", fmt.Errorf("failed to create address: %v", err)
	}
	
	return address.EncodeAddress(), nil
}

// generateAddressesFromSeed generates addresses from a seed phrase using different BIP standards
func generateAddressesFromSeed(seedPhrase string, count int) (map[string][]string, error) {
	// Validate seed phrase
	if !validateSeedPhrase(seedPhrase) {
		return nil, fmt.Errorf("invalid seed phrase")
	}
	
	// Convert seed phrase to seed
	seed := seedToRootKey(seedPhrase)
	
	// Generate addresses
	addresses := make(map[string][]string)
	addresses["BIP32 (Legacy)"] = make([]string, count)
	addresses["BIP44 (Legacy)"] = make([]string, count)
	addresses["BIP49 (SegWit-compatible)"] = make([]string, count)
	addresses["BIP84 (Native SegWit)"] = make([]string, count)
	
	// Show progress
	showLoadingBar(T("generating_addresses"), 2*time.Second)
	
	for i := 0; i < count; i++ {
		// BIP32 (Legacy)
		addr, err := deriveBip32Address(seed, i)
		if err != nil {
			return nil, fmt.Errorf("failed to derive BIP32 address: %v", err)
		}
		addresses["BIP32 (Legacy)"][i] = addr
		
		// BIP44 (Legacy)
		addr, err = deriveBip44Address(seed, 0, 0, i)
		if err != nil {
			return nil, fmt.Errorf("failed to derive BIP44 address: %v", err)
		}
		addresses["BIP44 (Legacy)"][i] = addr
		
		// BIP49 (SegWit-compatible)
		addr, err = deriveBip49Address(seed, 0, 0, i)
		if err != nil {
			return nil, fmt.Errorf("failed to derive BIP49 address: %v", err)
		}
		addresses["BIP49 (SegWit-compatible)"][i] = addr
		
		// BIP84 (Native SegWit)
		addr, err = deriveBip84Address(seed, 0, 0, i)
		if err != nil {
			return nil, fmt.Errorf("failed to derive BIP84 address: %v", err)
		}
		addresses["BIP84 (Native SegWit)"][i] = addr
	}
	
	return addresses, nil
}

// displayGeneratedAddresses displays generated addresses in a formatted way
func displayGeneratedAddresses(addresses map[string][]string) {
	// Prepare content for each BIP type
	var allContent []string
	
	for bipType, addrList := range addresses {
		// Add BIP type header
		allContent = append(allContent, whiteBold(amber(bipType)))
		allContent = append(allContent, green(safeRepeat("─", DISPLAY_WIDTH-4)))
		
		// Add addresses
		for i, addr := range addrList {
			indexStr := fmt.Sprintf("[%d]", i)
			allContent = append(allContent, fmt.Sprintf("%s %s", cyan(indexStr), addr))
		}
		
		// Add separator between BIP types
		if len(allContent) > 0 {
			allContent = append(allContent, green(safeRepeat("─", DISPLAY_WIDTH-4)))
		}
	}
	
	// Display the box
	fmt.Println(createBox(DISPLAY_WIDTH, T("generated_addresses"), allContent))
}

// checkAddressesBalances checks balances for a list of addresses
func checkAddressesBalances(addresses map[string][]string) {
	printInfo(T("checking_balances"))
	showLoadingBar(T("scanning_blockchain"), 3*time.Second)
	
	// Flatten the address list
	var allAddresses []struct {
		bipType string
		addr    string
	}
	
	for bipType, addrList := range addresses {
		for _, addr := range addrList {
			allAddresses = append(allAddresses, struct {
				bipType string
				addr    string
			}{bipType, addr})
		}
	}
	
	// Check each address
	foundFunds := false
	for i, addrInfo := range allAddresses {
		// Add a small delay between requests to avoid rate limiting
		if i > 0 {
			time.Sleep(1 * time.Second)
		}
		
		// Show progress
		fmt.Printf("%s %s\r", green("["+T("scan")+"]"), Tf("checking_address", addrInfo.bipType, addrInfo.addr))
		
		// Fetch wallet data
		data, err := fetchWalletData(addrInfo.addr)
		if err != nil {
			printError(fmt.Sprintf("Error checking address %s: %v", addrInfo.addr, err))
			continue
		}
		
		// Calculate balance
		balance, stats := calculateBalance(data)
		
		// Print summary
		if balance > 0 {
			foundFunds = true
			
			// Show a glitch for effect
			showGlitch()
			
			printSuccess(Tf("found_funds", addrInfo.bipType, addrInfo.addr))
			printSuccess(Tf("balance", balance))
			printInfo(T("fetching_details"))
			
			// Display detailed balance
			output := formatBalanceOutput(addrInfo.addr, balance, stats)
			fmt.Println(output)
			
			// Ask if user wants to see transactions
			reader := bufio.NewReader(os.Stdin)
			fmt.Print(amber(T("view_transactions") + " "))
			showTx, _ := reader.ReadString('\n')
			showTx = strings.TrimSpace(showTx)
			if strings.ToLower(showTx) == "y" {
				displayTransactions(addrInfo.addr, 5)
			}
		}
	}
	
	if !foundFunds {
		printWarning(T("no_funds_found"))
	}
}

// generateRandomMnemonic generates a random BIP39 mnemonic seed phrase
func generateRandomMnemonic() string {
	// Generate entropy (128 bits for 12 words)
	entropy, _ := bip39.NewEntropy(128)
	// Generate mnemonic from entropy
	mnemonic, _ := bip39.NewMnemonic(entropy)
	return mnemonic
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	b := make([]byte, length)
	cryptorand.Read(b)
	return hex.EncodeToString(b)[:length]
}

// displayInstanceStatus displays the status of all scanner instances
func displayInstanceStatus(instances []*InstanceStatus, stats *ScannerStats) {
	// Clear the display area
	clearLine()
	
	// Get terminal width
	width := getTerminalWidth()
	
	// Print header
	fmt.Println(green(safeRepeat("─", width)))
	
	// Print global stats
	walletsChecked, activeWalletsFound := stats.GetStats()
	fmt.Printf("%s %s\n", 
		green("["+T("stats")+"]"), 
		Tf("wallets_checked", walletsChecked, activeWalletsFound))
	
	// Print instance statuses
	for _, instance := range instances {
		status, address, bipType, endpoint := instance.GetStatus()
		
		// Format status with color
		var statusText string
		switch status {
		case "idle":
			statusText = cyan(T("instance_idle"))
		case "active":
			statusText = green(T("instance_active"))
		case "error":
			statusText = red(T("instance_error"))
		case "success":
			statusText = greenBold(T("instance_success"))
		default:
			statusText = white(status)
		}
		
		// Format address for display (truncate if needed)
		displayAddr := address
		if len(displayAddr) > 20 && status != "idle" {
			displayAddr = displayAddr[:10] + "..." + displayAddr[len(displayAddr)-7:]
		}
		
		// Format the status line
		var statusLine string
		if status == "idle" {
			statusLine = Tf("instance_status", instance.ID, statusText)
		} else {
			walletInfo := Tf("checking_wallet", displayAddr, endpoint)
			if len(walletInfo) > width-30 {
				walletInfo = walletInfo[:width-33] + "..."
			}
			statusLine = fmt.Sprintf("%s | %s | %s", 
				Tf("instance_status", instance.ID, statusText),
				bipType,
				walletInfo)
		}
		
		fmt.Println(statusLine)
	}
	
	fmt.Println(green(safeRepeat("─", width)))
}

// scanWalletInstance is a function that runs in a goroutine to scan random wallets
// and save active ones to a CSV file
func scanWalletInstance(instanceID int, stats *ScannerStats, instanceStatus *InstanceStatus, done <-chan bool) {
	// Generate a unique CSV filename with date, hour, instance number, and random characters
	now := time.Now()
	randomStr := generateRandomString(10)
	csvFilename := fmt.Sprintf("%s_%d_%s.csv", 
		now.Format("20060102_1504"), 
		instanceID,
		randomStr)
	
	// Create CSV file with headers
	file, err := os.Create(csvFilename)
	if err != nil {
		instanceStatus.SetError(fmt.Sprintf("Failed to create CSV file: %v", err))
		return
	}
	
	writer := csv.NewWriter(file)
	writer.Write([]string{
		"Timestamp",
		T("seed_phrase"),
		"Wallet Type",
		T("path"),
		T("address"),
		"Balance (BTC)",
		T("transactions"),
		"Explorer URL",
	})
	writer.Flush()
	file.Close()
	
	// Print instance started message
	printInfo(Tf("instance_started", instanceID, csvFilename))
	
	// Set initial status
	instanceStatus.Update("idle", "", "", "")
	
	// Run scanner loop
	for {
		select {
		case <-done:
			return
		default:
			// Generate a random seed phrase
			seedPhrase := generateRandomMnemonic()
			
			// Convert to seed
			seed := seedToRootKey(seedPhrase)
			
			// Generate addresses for different BIP standards
			// We'll only check the first address (index 0) for each standard to speed up scanning
			var addresses []struct {
				walletType string
				address    string
				path       string
			}
			
			// BIP32 (Legacy)
			bip32Addr, err := deriveBip32Address(seed, 0)
			if err == nil {
				addresses = append(addresses, struct {
					walletType string
					address    string
					path       string
				}{"BIP32 (Legacy)", bip32Addr, "m/0'/0/0"})
			}
			
			// BIP44 (Legacy)
			bip44Addr, err := deriveBip44Address(seed, 0, 0, 0)
			if err == nil {
				addresses = append(addresses, struct {
					walletType string
					address    string
					path       string
				}{"BIP44 (Legacy)", bip44Addr, "m/44'/0'/0'/0/0"})
			}
			
			// BIP49 (SegWit-compatible)
			bip49Addr, err := deriveBip49Address(seed, 0, 0, 0)
			if err == nil {
				addresses = append(addresses, struct {
					walletType string
					address    string
					path       string
				}{"BIP49 (SegWit-compatible)", bip49Addr, "m/49'/0'/0'/0/0"})
			}
			
			// BIP84 (Native SegWit)
			bip84Addr, err := deriveBip84Address(seed, 0, 0, 0)
			if err == nil {
				addresses = append(addresses, struct {
					walletType string
					address    string
					path       string
				}{"BIP84 (Native SegWit)", bip84Addr, "m/84'/0'/0'/0/0"})
			}
			
			// Update counter
			stats.IncrementWalletsChecked()
			
			// Check each address
			for _, addrInfo := range addresses {
				// Update status to show we're checking this address
				endpoint := getCurrentEndpoint()
				instanceStatus.Update("active", addrInfo.address, addrInfo.walletType, endpoint)
				
				// Fetch wallet data
				data, err := fetchWalletData(addrInfo.address)
				if err != nil {
					instanceStatus.SetError(fmt.Sprintf("Error: %v", err))
					continue
				}
				
				// Calculate balance
				balance, walletStats := calculateBalance(data)
				txCount := walletStats.TxCount
				
				// Check if address is active
				if balance > 0 || txCount > 0 {
					// Update status to show success
					instanceStatus.Update("success", addrInfo.address, addrInfo.walletType, endpoint)
					
					stats.IncrementActiveWalletsFound()
					
					// Format balance
					balanceBtc := fmt.Sprintf("%.8f", balance)
					
					// Create explorer URL
					explorerURL := fmt.Sprintf("https://mempool.space/address/%s", addrInfo.address)
					
					// Print success message with retro styling
					fmt.Printf("\n%s [Instance #%d] %s: %s\n", 
						greenBold("["+T("active_wallet_found")+"]"), 
						instanceID,
						addrInfo.walletType, 
						addrInfo.address)
					
					// Save to CSV
					timestamp := time.Now().Format("2006-01-02 15:04:05")
					file, err := os.OpenFile(csvFilename, os.O_APPEND|os.O_WRONLY, 0644)
					if err == nil {
						writer := csv.NewWriter(file)
						writer.Write([]string{
							timestamp,
							seedPhrase,
							addrInfo.walletType,
							addrInfo.path,
							addrInfo.address,
							balanceBtc,
							fmt.Sprintf("%d", txCount),
							explorerURL,
						})
						writer.Flush()
						file.Close()
					}
					
					// Reset status after a brief pause
					time.Sleep(2 * time.Second)
					instanceStatus.Update("idle", "", "", "")
				} else {
					// Reset status to idle
					instanceStatus.Update("idle", "", "", "")
				}
			}
			
			// Small delay to avoid overwhelming the API
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// runMultiInstanceScanner runs multiple instances of the wallet scanner
func runMultiInstanceScanner(instanceCount int) {
	// Show retro banner
	var content []string
	content = append(content, "")
	content = append(content, T("scanner_warning"))
	content = append(content, "")
	
	fmt.Println(createBox(MENU_HEADER_WIDTH, T("multi_scanner"), content))
	
	printInfo(Tf("starting_multi_scanner", instanceCount))
	
	// Create a shared statistics tracker
	stats := &ScannerStats{
		WalletsChecked:    0,
		ActiveWalletsFound: 0,
	}
	
	// Create instance status trackers
	instances := make([]*InstanceStatus, instanceCount)
	for i := 0; i < instanceCount; i++ {
		instances[i] = &InstanceStatus{
			ID:     i + 1,
			Status: "idle",
		}
	}
	
	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	
	// Create a done channel to signal all goroutines to stop
	done := make(chan bool)
	
	// Start the scanner instances
	for i := 0; i < instanceCount; i++ {
		go scanWalletInstance(i+1, stats, instances[i], done)
	}
	
	// Start a goroutine to periodically update the display
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				displayInstanceStatus(instances, stats)
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()
	
	// Wait for interrupt signal
	<-c
	
	// Signal all goroutines to stop
	close(done)
	
	// Give goroutines a moment to clean up
	time.Sleep(500 * time.Millisecond)
	
	// Print final statistics
	walletsChecked, activeWalletsFound := stats.GetStats()
	fmt.Println("\n" + T("stopping_all_scanners"))
	fmt.Printf("%s: %d\n", T("total_wallets_checked"), walletsChecked)
	fmt.Printf("%s: %d\n", T("total_active_wallets"), activeWalletsFound)
}

// scanRandomWallets runs a single instance of the wallet scanner
func scanRandomWallets() {
	// Show retro banner
	var content []string
	content = append(content, "")
	content = append(content, T("scanner_warning"))
	content = append(content, "")
	
	fmt.Println(createBox(MENU_HEADER_WIDTH, T("random_wallet_scanner"), content))
	
	// Generate a unique CSV filename with date, hour, and random characters
	now := time.Now()
	randomStr := generateRandomString(10)
	csvFilename := fmt.Sprintf("wallets_%s_%s.csv", 
		now.Format("20060102_1504"), 
		randomStr)
	
	// Create CSV file with headers
	file, err := os.Create(csvFilename)
	if err != nil {
		printError(fmt.Sprintf("Failed to create CSV file: %v", err))
		return
	}
	
	writer := csv.NewWriter(file)
	writer.Write([]string{
		"Timestamp",
		T("seed_phrase"),
		"Wallet Type",
		T("path"),
		T("address"),
		"Balance (BTC)",
		T("transactions"),
		"Explorer URL",
	})
	writer.Flush()
	file.Close()
	
	printInfo(Tf("starting_scanner", csvFilename))
	printInfo(T("press_ctrl_c"))
	
	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	
	// Initialize counters
	walletsChecked := 0
	activeWalletsFound := 0
	
	// Start scanning loop
	scannerRunning := true
	for scannerRunning {
		select {
		case <-c:
			scannerRunning = false
		default:
			// Generate a random seed phrase
			seedPhrase := generateRandomMnemonic()
			
			// Convert to seed
			seed := seedToRootKey(seedPhrase)
			
			// Generate addresses for different BIP standards
			// We'll only check the first address (index 0) for each standard to speed up scanning
			var addresses []struct {
				walletType string
				address    string
				path       string
			}
			
			// BIP32 (Legacy)
			bip32Addr, err := deriveBip32Address(seed, 0)
			if err == nil {
				addresses = append(addresses, struct {
					walletType string
					address    string
					path       string
				}{"BIP32 (Legacy)", bip32Addr, "m/0'/0/0"})
			}
			
			// BIP44 (Legacy)
			bip44Addr, err := deriveBip44Address(seed, 0, 0, 0)
			if err == nil {
				addresses = append(addresses, struct {
					walletType string
					address    string
					path       string
				}{"BIP44 (Legacy)", bip44Addr, "m/44'/0'/0'/0/0"})
			}
			
			// BIP49 (SegWit-compatible)
			bip49Addr, err := deriveBip49Address(seed, 0, 0, 0)
			if err == nil {
				addresses = append(addresses, struct {
					walletType string
					address    string
					path       string
				}{"BIP49 (SegWit-compatible)", bip49Addr, "m/49'/0'/0'/0/0"})
			}
			
			// BIP84 (Native SegWit)
			bip84Addr, err := deriveBip84Address(seed, 0, 0, 0)
			if err == nil {
				addresses = append(addresses, struct {
					walletType string
					address    string
					path       string
				}{"BIP84 (Native SegWit)", bip84Addr, "m/84'/0'/0'/0/0"})
			}
			
			// Update counter
			walletsChecked++
			
			// Check each address
			for _, addrInfo := range addresses {
				// Show progress
				fmt.Printf("\r%s %s", green("["+T("scan")+"]"), Tf("testing_address", addrInfo.address, addrInfo.walletType))
				
				// Fetch wallet data
				data, err := fetchWalletData(addrInfo.address)
				if err != nil {
					continue
				}
				
				// Calculate balance
				balance, walletStats := calculateBalance(data)
				txCount := walletStats.TxCount
				
				// Check if address is active
				if balance > 0 || txCount > 0 {
					activeWalletsFound++
					
					// Format balance
					balanceBtc := fmt.Sprintf("%.8f", balance)
					
					// Create explorer URL
					explorerURL := fmt.Sprintf("https://mempool.space/address/%s", addrInfo.address)
					
					// Print success message with retro styling
					fmt.Printf("\n%s %s: %s\n", 
						greenBold("["+T("active_wallet_found")+"]"), 
						addrInfo.walletType, 
						addrInfo.address)
					
					// Save to CSV
					timestamp := time.Now().Format("2006-01-02 15:04:05")
					file, err := os.OpenFile(csvFilename, os.O_APPEND|os.O_WRONLY, 0644)
					if err == nil {
						writer := csv.NewWriter(file)
						writer.Write([]string{
							timestamp,
							seedPhrase,
							addrInfo.walletType,
							addrInfo.path,
							addrInfo.address,
							balanceBtc,
							fmt.Sprintf("%d", txCount),
							explorerURL,
						})
						writer.Flush()
						file.Close()
					}
				}
			}
			
			// Show stats
			fmt.Printf("\r%s %s", green("["+T("stats")+"]"), Tf("wallets_checked", walletsChecked, activeWalletsFound))
			
			// Small delay to avoid overwhelming the API
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	// Print final statistics
	fmt.Println("\n" + T("stopping_scanner"))
	fmt.Printf("%s: %d\n", T("total_wallets_checked"), walletsChecked)
	fmt.Printf("%s: %d\n", T("total_active_wallets"), activeWalletsFound)
	fmt.Printf("%s: %s\n", T("results_saved"), csvFilename)
}

// showLanguageSelection displays the language selection menu
func showLanguageSelection() {
	clearScreen()
	
	// Show a matrix effect for visual appeal
	showMatrixEffect(2 * time.Second)
	
	// Prepare content for language selection
	var content []string
	content = append(content, T("select_language"))
	content = append(content, "")
	content = append(content, fmt.Sprintf("1. %s", T("english")))
	content = append(content, fmt.Sprintf("2. %s", T("chinese")))
	content = append(content, fmt.Sprintf("3. %s", T("japanese")))
	content = append(content, fmt.Sprintf("4. %s", T("russian")))
	
	// Display the box
	fmt.Println(createBox(DISPLAY_WIDTH, T("language_selection"), content))
	
	// Get user input
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(amber(T("enter_command") + " "))
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	
	// Set language based on choice
	switch choice {
	case "1":
		currentLanguage = LANG_ENGLISH
		printSuccess(Tf("language_selected", T("english")))
	case "2":
		currentLanguage = LANG_CHINESE
		printSuccess(Tf("language_selected", T("chinese")))
	case "3":
		currentLanguage = LANG_JAPANESE
		printSuccess(Tf("language_selected", T("japanese")))
	case "4":
		currentLanguage = LANG_RUSSIAN
		printSuccess(Tf("language_selected", T("russian")))
	default:
		// Default to English if invalid choice
		currentLanguage = LANG_ENGLISH
		printWarning(Tf("language_selected", T("english")))
	}
	
	// Small delay to show the language selection message
	time.Sleep(1 * time.Second)
}

// showMainMenu displays the main menu
func showMainMenu() {
	clearScreen()
	
	// Print logo with typing effect
	for _, line := range strings.Split(LOGO, "\n") {
		fmt.Println(green(line))
		time.Sleep(50 * time.Millisecond)
	}
	
	// Prepare content for menu header
	var headerContent []string
	headerContent = append(headerContent, "")
	headerContent = append(headerContent, T("welcome_message"))
	headerContent = append(headerContent, "")
	headerContent = append(headerContent, T("risk_warning"))
	
	// Display menu header
	fmt.Println(createBox(MENU_HEADER_WIDTH, T("app_title"), headerContent))
	
	// Print menu options with typing effect
	typeText(greenBold("    [1] " + T("menu_check_balance")), 20*time.Millisecond)
	typeText(greenBold("    [2] " + T("menu_view_transactions")), 20*time.Millisecond)
	typeText(greenBold("    [3] " + T("menu_generate_addresses")), 20*time.Millisecond)
	typeText(greenBold("    [4] " + T("menu_scan_wallets")), 20*time.Millisecond)
	typeText(greenBold("    [5] " + T("menu_multi_scan")), 20*time.Millisecond)
	typeText(greenBold("    [6] " + T("language_selection")), 20*time.Millisecond)
	typeText(greenBold("    [7] " + T("menu_exit")), 20*time.Millisecond)
	
	// Prepare content for menu footer
	var footerContent []string
	footerContent = append(footerContent, "  [X] " + T("disconnect"))
	
	// Display menu footer
	fmt.Println(createBox(MENU_HEADER_WIDTH, "", footerContent))
}

func main() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
	
	// Show language selection at startup
	showLanguageSelection()
	
	// Test and sort endpoints
	testAndSortEndpoints()
	
	// Show main menu
	showMainMenu()
	
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(amber(T("enter_command") + " "))
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	
	switch choice {
	case "1":
		// Check wallet balance
		clearScreen()
		fmt.Println(createBox(MENU_HEADER_WIDTH, T("menu_check_balance"), []string{""}))
		
		fmt.Print(amber(T("enter_address") + " "))
		address, _ := reader.ReadString('\n')
		address = strings.TrimSpace(address)
		if address != "" {
			checkWalletBalance(address)
		} else {
			printError(T("no_address"))
		}
		
	case "2":
		// View transaction history
		clearScreen()
		fmt.Println(createBox(MENU_HEADER_WIDTH, T("menu_view_transactions"), []string{""}))
		
		fmt.Print(amber(T("enter_address") + " "))
		address, _ := reader.ReadString('\n')
		address = strings.TrimSpace(address)
		if address != "" {
			fmt.Print(amber(T("enter_tx_limit") + " "))
			limitStr, _ := reader.ReadString('\n')
			limitStr = strings.TrimSpace(limitStr)
			limit := 5
			if limitStr != "" {
				if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
					limit = l
				}
			}
			displayTransactions(address, limit)
		} else {
			printError(T("no_address"))
		}
		
	case "3":
		// Generate addresses from seed phrase
		clearScreen()
		fmt.Println(createBox(MENU_HEADER_WIDTH, T("menu_generate_addresses"), []string{""}))
		
		fmt.Print(amber(T("enter_seed_phrase") + " "))
		seedPhrase, _ := reader.ReadString('\n')
		seedPhrase = strings.TrimSpace(seedPhrase)
		if seedPhrase != "" {
			// Validate seed phrase
			if !validateSeedPhrase(seedPhrase) {
				printError(T("invalid_seed_phrase"))
				break
			}
			
			// Get number of addresses to generate
			fmt.Print(amber(T("enter_address_count") + " "))
			countStr, _ := reader.ReadString('\n')
			
			countStr = strings.TrimSpace(countStr)
			count := 5
			if countStr != "" {
				if c, err := strconv.Atoi(countStr); err == nil && c > 0 {
					count = c
				}
			}
			
			printInfo(Tf("generating_addresses_msg", count))
			addresses, err := generateAddressesFromSeed(seedPhrase, count)
			if err != nil {
				printError(fmt.Sprintf("Error generating addresses: %v", err))
				break
			}
			
			// Display generated addresses
			displayGeneratedAddresses(addresses)
			
			// Ask if user wants to check balances
			fmt.Print(amber(T("check_balances") + " "))
			checkBalance, _ := reader.ReadString('\n')
			checkBalance = strings.TrimSpace(checkBalance)
			if strings.ToLower(checkBalance) == "y" {
				checkAddressesBalances(addresses)
			}
		} else {
			printError(T("no_seed_phrase"))
		}
		
	case "4":
		// Scan random wallets
		clearScreen()
		scanRandomWallets()
		
	case "5":
		// Multi-instance wallet scanner
		clearScreen()
		fmt.Println(createBox(MENU_HEADER_WIDTH, T("menu_multi_scan"), []string{""}))
		
		fmt.Print(amber(T("enter_instance_count") + " "))
		countStr, _ := reader.ReadString('\n')
		countStr = strings.TrimSpace(countStr)
		
		instanceCount := 4 // Default to 4 instances
		if countStr != "" {
			if count, err := strconv.Atoi(countStr); err == nil && count > 0 {
				instanceCount = count
			}
		}
		
		runMultiInstanceScanner(instanceCount)
		
	case "6":
		// Language selection
		showLanguageSelection()
		main() // Return to main menu after language selection
		return
		
	case "7":
		// Exit
		clearScreen()
		typeText(green(T("disconnecting")), 30*time.Millisecond)
		showLoadingBar(T("clearing_traces"), 2*time.Second)
		typeText(green(T("connection_terminated")), 30*time.Millisecond)
		os.Exit(0)
		
	default:
		printError(T("invalid_command"))
		time.Sleep(2 * time.Second)
		main() // Restart the menu
		return
	}
	
	// Return to main menu after operation completes
	fmt.Print(amber("\n" + T("press_enter")))
	reader.ReadString('\n')
	main()
}
