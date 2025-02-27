package main

import (
	"fmt"
	"bufio"
	"flag"
	"os"
	"sync"
	"github.com/logrusorgru/aurora"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"io/ioutil"
	"time"
	"strings"
	"regexp"
	)

func init() {
	flag.Usage = func() {
		h := []string{
			"",
			"xKeys (eXtract Keys from source)",
			"",
			"By : viloid [Sec7or - Surabaya Hacker Link]",
			"",
			"Basic Usage :",
			" ▶ echo http://domain.com/path/file.js | xkeys",
			" ▶ cat listurls.txt | xkeys -o cans.txt",
			"",
			"Options :",
			"  -w, --wordlist <wordlist>             Custom wordlists keyword pattern",
			"  -k, --key <keyword>                   Custom single keyword pattern",
			"  -H, --header <header>                 Header to the request",
			"  -o, --output <output>                 Output file (*default xkeys.txt)",
			"  -x, --proxy <proxy>                   HTTP proxy",
			"  -t, --telegram <BotToken|RecipientID> Telegram notification",
			"",
			"",
		}
		fmt.Fprintf(os.Stderr, strings.Join(h, "\n"))
	}
}

func main() {

	var headers headerArgs
	flag.Var(&headers, "header", "")
	flag.Var(&headers, "H", "")

	var outputFile string
	flag.StringVar(&outputFile, "output", "xkeys.txt", "")
	flag.StringVar(&outputFile, "o", "xkeys.txt", "")

	var proxy string
	flag.StringVar(&proxy, "proxy", "", "")
	flag.StringVar(&proxy, "x", "", "")

	var wlist string
	flag.StringVar(&wlist, "wordlist", "", "")
	flag.StringVar(&wlist, "w", "", "")

	var wkey string
	flag.StringVar(&wkey, "key", "", "")
	flag.StringVar(&wkey, "k", "", "")

	var notif string
	flag.StringVar(&notif, "telegram", "", "")
	flag.StringVar(&notif, "t", "", "")

	flag.Parse()

	client := newClient(proxy)
	var wg sync.WaitGroup

	sc := bufio.NewScanner(os.Stdin)

	for sc.Scan() {
		u := sc.Text()
		wg.Add(1)

		go func() {

			defer wg.Done()

			req, err := http.NewRequest("GET", u, nil)

			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create request: %s\n", err)
				return
			}

			if headers == nil {
				req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; xkeys/0.1; +https://github.com/vsec7/xkeys)")
			}
			
			// add headers to the request
			for _, h := range headers {
				parts := strings.SplitN(h, ":", 2)

				if len(parts) != 2 {
					continue
				}
				req.Header.Set(parts[0], parts[1])
			}

			// send the request
			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "request failed: %s\n", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				data, _ := ioutil.ReadAll(resp.Body)
				
				var wl []string
				switch {
				case len(wlist) != 0 :
					f, err := ioutil.ReadFile(wlist)
					if err != nil {
						fmt.Printf("[%s] File : %s Not Exist!\n\n", aurora.Red("ERROR"), aurora.Red(wlist))
						os.Exit(1)
					}					
					wl = strings.Split(string(f), "\n")
				case len(wkey) != 0 :
					wl = []string{wkey}
				default :
					wl = keywords()
				}

				ExtractKeys( u, string(data), outputFile, wl, notif)

			} else {
				fmt.Printf("[%d] %s | %s\n", aurora.Red(resp.StatusCode), u, aurora.Red("Nothing!"))
			}
		}()
	}
	wg.Wait()
}

func ExtractKeys( u string, f string, o string, w []string, n string) {	
	out := make([][]string, 0)

	for _, keys := range w {
		var re = regexp.MustCompile(`(?i)`+keys+`['\"]?\s?(=|:)\s?['\"]?([^\s"'&]+)`)
		for _, q := range re.FindAllStringSubmatch(f, -1){			
			var arr = []string{keys,q[2]}
			out = append(out, arr)
		}		
	}

	if len(out) != 0 {
		fmt.Printf("[%s] %s | %s\n", aurora.Green("200"), aurora.Magenta(u), aurora.Green("Found!"))
		file, err := os.OpenFile(o, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Failed Creating File: %s", err)
		}
		buf := bufio.NewWriter(file)
		buf.WriteString("[+] "+u+"\n")

		for _, opt := range out {
			fmt.Printf("\t[%s] %s\n", aurora.Red(opt[0]), aurora.Green(opt[1]))
			buf.WriteString("\t["+opt[0]+"] "+opt[1]+"\n")

			if len(n) != 0 {
				conf := strings.Split(string(n), "|")
				Tele(conf[0], conf[1], "[ xKeys ]\n\n[+] "+u+"\n\n["+opt[0]+"] : "+opt[1]+" \n")
			}
		}
		buf.Flush()
		file.Close()
	} else {
		fmt.Printf("[%s] %s | %s\n", aurora.Green("200"), aurora.Magenta(u), aurora.Red("Nothing!"))
	}
}

func Tele( token string, targetid string, txt string) {
	client := newClient("")
	req, err := http.NewRequest("GET", "https://api.telegram.org/bot"+token+"/sendMessage?chat_id="+targetid+"&text="+url.QueryEscape(txt), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create request: %s\n", err)
		return
	}
	client.Do(req)
}

func newClient(proxy string) *http.Client {
	tr := &http.Transport{
		MaxIdleConns:		30,
		IdleConnTimeout:	time.Second,
		TLSClientConfig:	&tls.Config{InsecureSkipVerify: true},
		DialContext:		(&net.Dialer{
			Timeout:	time.Second * 10,
			KeepAlive:	time.Second,
		}).DialContext,
	}

	if proxy != "" {
		if p, err := url.Parse(proxy); err == nil {
			tr.Proxy = http.ProxyURL(p)
		}
	}

	re := func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return &http.Client{
		Transport:		tr,
		CheckRedirect: 	re,
		Timeout:		time.Second * 10,
	}
}

type headerArgs []string

func (h *headerArgs) Set(val string) error {
	*h = append(*h, val)
	return nil
}

func (h headerArgs) String() string {
	return "string"
}

func keywords() []string {
	return []string{
		"access_key",
		"access_token",
		"accessKey",
		"accessToken",
		"account_sid",
		"accountsid",
		"admin_pass",
		"admin_user",
		"api_key",
		"api_secret",
		"apikey",
		"app_debug",
		"app_env",
		"app_id",
		"app_key",
		"app_log_level",
		"app_name",
		"app_secret",
		"app_url",
		"application_id",
		"aws_secret_token",
		"auth",
		"authsecret",
		"aws_access",
		"aws_access_key_id",
		"aws_bucket",
		"aws_config",
		"aws_default_region",
		"aws_key",
		"aws_secret",
		"aws_secret_access_key",
		"aws_secret_key",
		"aws_token",
		"broadcast_driver",
		"bucket_password",
		"cache_driver",
		"client_secret",
		"cloudinary_api_key",
		"cloudinary_api_secret",
		"cloudinary_name",
		"connectionstring",
		"consumer_secret",
		"credentials",
		"database_dialect",
		"database_host",
		"database_logging",
		"database_password",
		"database_schema",
		"database_schema_test",
		"database_url",
		"database_username",
		"db_connection",
		"db_database",
		"db_dialect",
		"db_host",
		"db_password",
		"db_port",
		"db_server",
		"db_username",
		"dbpasswd",
		"dbpassword",
		"dbuser",
		"debug",
		"django_password",
		"dotfiles",
		"elastica_host",
		"elastica_port",
		"elastica_prefix",
		"email_host_password",
		"env",
		"facebook_app_secret",
		"facebook_secret",
		"fb_app_secret",
		"fb_id",
		"fb_secret",
		"gatsby_wordpress_base_url",
		"gatsby_wordpress_client_id",
		"gatsby_wordpress_client_secret",
		"gatsby_wordpress_password",
		"gatsby_wordpress_protocol",
		"gatsby_wordpress_user",
		"github_id",
		"github_secret",
		"google_id",
		"google_oauth",
		"google_oauth_client_id",
		"google_oauth_client_secret",
		"google_oauth_secret",
		"google_secret",
		"google_server_key",
		"gsecr",
		"heroku_api_key",
		"heroku_key",
		"heroku_oauth",
		"heroku_oauth_secret",
		"heroku_oauth_token",
		"heroku_secret",
		"heroku_secret_token",
		"htaccess_pass",
		"htaccess_user",
		"incident_bot_name",
		"incident_channel_name",
		"jwt_passphrase",
		"jwt_password",
		"jwt_public_key",
		"jwt_secret",
		"jwt_secret_key",
		"jwt_secret_token",
		"jwt_token",
		"jwt_user",
		"keyPassword",
		"location_hostname",
		"location_protocol",
		"log_channel",
		"mail_driver",
		"mail_encryption",
		"mail_from_address",
		"mail_from_name",
		"mail_host",
		"mail_password",
		"mail_port",
		"mail_username",
		"mailgun_key",
		"mailgun_secret",
		"mix_pusher_app_cluster",
		"mix_pusher_app_key",
		"mysql_password",
		"node_env",
		"oauth",
		"oauth_discord_id",
		"oauth_discord_secret",
		"oauth_key",
		"oauth_token",
		"oauth2_secret",
		"password",
		"paypal_identity_token",
		"paypal_sandbox",
		"paypal_secret",
		"paypal_token",
		"playbooks_url",
		"postgres_password",
		"private",
		"private_key",
		"pusher_app_cluster",
		"pusher_app_id",
		"pusher_app_key",
		"pusher_app_secret",
		"queue_connection",
		"queue_driver",
		"redis_host",
		"redis_password",
		"redis_port",
		"response_auth_jwt_secret",
		"response_data_secret",
		"response_data_url",
		"root_password",
		"sa_password",
		"s3_access_key_id",
		"s3_secret_key",
		"secret",
		"secret_access_key",
		"secret_bearer",
		"secret_key",
		"secret_token",
		"secretKey",
		"security_credentials",
		"send_keys",
		"sentry_dsn",
		"session_driver",
		"session_lifetime",
		"sf_username",
		"sid twilio",
		"sid_token",
		"sid_twilio",
		"slack_channel",
		"slack_incoming_webhook",
		"slack_key",
		"slack_outgoing_token",
		"slack_secret",
		"slack_signing_secret",
		"slack_token",
		"slack_url",
		"slack_webhook",
		"slack_webhook_url",
		"square_access_token",
		"square_apikey",
		"square_app",
		"square_app_id",
		"square_appid",
		"square_secret",
		"square_token",
		"squareSecret",
		"squareToken",
		"ssh",
		"ssh2_auth_password",
		"sshkey",
		"staging",
		"status_page",
		"status_page_playbook_url",
		"storePassword",
		"strip_key",
		"strip_secret",
		"strip_secret_token",
		"strip_token",
		"stripe_key",
		"stripe_secret",
		"stripe_secret_token",
		"stripe_token",
		"stripSecret",
		"stripToken",
		"token_twilio",
		"trusted_hosts",
		"trusted_proxies",
		"twi_auth",
		"twi_sid",
		"twilio_account_id",
		"twilio_account_secret",
		"twilio_account_sid",
		"twilio_accountsid",
		"twilio_acount_sid NOT env",
		"twilio_api",
		"twilio_api_auth",
		"twilio_api_key",
		"twilio_api_secret",
		"twilio_api_sid",
		"twilio_api_token",
		"twilio_auth",
		"twilio_auth_token",
		"twilio_secret",
		"twilio_secret_token",
		"twilio_sid",
		"TWILIO_SID NOT env",
		"twilio_token",
		"twilioapiauth",
		"twilioapisecret",
		"twilioapisid",
		"twilioapitoken",
		"TwilioAuthKey",
		"TwilioAuthSid",
		"twilioauthtoken",
		"TwilioKey",
		"twiliosecret",
		"TwilioSID",
		"twiliotoken",
		"twitter_api_secret",
		"twitter_consumer_key",
		"twitter_consumer_secret",
		"twitter_key",
		"twitter_secret",
		"twitter_token",
		"twitterKey",
		"twitterSecret",
		"wordpress_password",
		"zen_key",
		"zen_tkn",
		"zen_token",
		"zendesk_api_token",
		"zendesk_key",
		"zendesk_token",
		"zendesk_url",
		"zendesk_username",
	}
}
